"""
Multi-part parsing for file uploads.

Exposes one class, ``MultiPartParser``, which feeds chunks of uploaded data to
file upload handlers for processing.
"""

import base64
import binascii
import collections
import html

from django.conf import settings
from django.core.exceptions import (
    RequestDataTooBig,
    SuspiciousMultipartForm,
    TooManyFieldsSent,
    TooManyFilesSent,
)
from django.core.files.uploadhandler import SkipFile, StopFutureHandlers, StopUpload
from django.utils.datastructures import MultiValueDict
from django.utils.encoding import force_str
from django.utils.http import parse_header_parameters
from django.utils.regex_helper import _lazy_re_compile

__all__ = ("MultiPartParser", "MultiPartParserError", "InputStreamExhausted")


class MultiPartParserError(Exception):
    pass


class InputStreamExhausted(Exception):
    """
    No more reads are allowed from this device.
    """

    pass


RAW = "raw"
FILE = "file"
FIELD = "field"
FIELD_TYPES = frozenset([FIELD, RAW])
MAX_TOTAL_HEADER_SIZE = 1024


class MultiPartParser:
    """
    An RFC 7578 multipart/form-data parser.

    ``MultiValueDict.parse()`` reads the input stream in ``chunk_size`` chunks
    and returns a tuple of ``(MultiValueDict(POST), MultiValueDict(FILES))``.
    """

    boundary_re = _lazy_re_compile(r"[ -~]{0,200}[!-~]")

    def __init__(self, META, input_data, upload_handlers, encoding=None):
        """
        Initialize the MultiPartParser object.

        :META:
            The standard ``META`` dictionary in Django request objects.
        :input_data:
            The raw post data, as a file-like object.
        :upload_handlers:
            A list of UploadHandler instances that perform operations on the
            uploaded data.
        :encoding:
            The encoding with which to treat the incoming data.
        """
        # Content-Type should contain multipart and the boundary information.
        content_type = META.get("CONTENT_TYPE", "")
        if not content_type.startswith("multipart/"):
            raise MultiPartParserError("Invalid Content-Type: %s" % content_type)

        try:
            content_type.encode("ascii")
        except UnicodeEncodeError:
            raise MultiPartParserError(
                "Invalid non-ASCII Content-Type in multipart: %s"
                % force_str(content_type)
            )

        # Parse the header to get the boundary to split the parts.
        _, opts = parse_header_parameters(content_type)
        boundary = opts.get("boundary")
        if not boundary or not self.boundary_re.fullmatch(boundary):
            raise MultiPartParserError(
                "Invalid boundary in multipart: %s" % force_str(boundary)
            )

        # Content-Length should contain the length of the body we are about
        # to receive.
        try:
            content_length = int(META.get("CONTENT_LENGTH", 0))
        except (ValueError, TypeError):
            content_length = 0

        if content_length < 0:
            # This means we shouldn't continue...raise an error.
            raise MultiPartParserError("Invalid content length: %r" % content_length)

        self._boundary = boundary.encode("ascii")
        self._input_data = input_data

        # For compatibility with low-level network APIs (with 32-bit integers),
        # the chunk size should be < 2^31, but still divisible by 4.
        possible_sizes = [x.chunk_size for x in upload_handlers if x.chunk_size]
        self._chunk_size = min([2**31 - 4, *possible_sizes])

        self._meta = META
        self._encoding = encoding or settings.DEFAULT_CHARSET
        self._content_length = content_length
        self._upload_handlers = upload_handlers

    def parse(self):
        # Call the actual parse routine and close all open files in case of
        # errors. This is needed because if exceptions are thrown the
        # MultiPartParser will not be garbage collected immediately and
        # resources would be kept alive. This is only needed for errors because
        # the Request object closes all uploaded files at the end of the
        # request.
        try:
            return self._parse()
        except Exception:
            if hasattr(self, "_files"):
                for _, files in self._files.lists():
                    for fileobj in files:
                        fileobj.close()
            raise

    def _parse(self):
        """
        Parse the POST data and break it into a FILES MultiValueDict and a POST
        MultiValueDict.

        Return a tuple containing the POST and FILES dictionary, respectively.
        """
        from django.http import QueryDict

        encoding = self._encoding
        handlers = self._upload_handlers

        # HTTP spec says that Content-Length >= 0 is valid
        # handling content-length == 0 before continuing
        if self._content_length == 0:
            return QueryDict(encoding=self._encoding), MultiValueDict()

        # See if any of the handlers take care of the parsing.
        # This allows overriding everything if need be.
        for handler in handlers:
            result = handler.handle_raw_input(
                self._input_data,
                self._meta,
                self._content_length,
                self._boundary,
                encoding,
            )
            # Check to see if it was handled
            if result is not None:
                return result[0], result[1]

        # Create the data structures to be used later.
        self._post = QueryDict(mutable=True)
        self._files = MultiValueDict()

        # Instantiate the parser and stream:
        stream = LazyStream(ChunkIter(self._input_data, self._chunk_size))

        # Whether or not to signal a file-completion at the beginning of the
        # loop.
        old_field_name = None
        counters = [0] * len(handlers)

        # Number of bytes that have been read.
        num_bytes_read = 0
        # To count the number of keys in the request.
        num_post_keys = 0
        # To count the number of files in the request.
        num_files = 0
        # To limit the amount of data read from the request.
        read_size = None
        # Whether a file upload is finished.
        uploaded_file = True

        try:
            for item_type, meta_data, field_stream in Parser(stream, self._boundary):
                if old_field_name:
                    # We run this at the beginning of the next loop
                    # since we cannot be sure a file is complete until
                    # we hit the next boundary/part of the multipart content.
                    self.handle_file_complete(old_field_name, counters)
                    old_field_name = None
                    uploaded_file = True

                if (
                    item_type in FIELD_TYPES
                    and settings.DATA_UPLOAD_MAX_NUMBER_FIELDS is not None
                ):
                    # Avoid storing more than DATA_UPLOAD_MAX_NUMBER_FIELDS.
                    num_post_keys += 1
                    # 2 accounts for empty raw fields before and after the
                    # last boundary.
                    if settings.DATA_UPLOAD_MAX_NUMBER_FIELDS + 2 < num_post_keys:
                        raise TooManyFieldsSent(
                            "The number of GET/POST parameters exceeded "
                            "settings.DATA_UPLOAD_MAX_NUMBER_FIELDS."
                        )

                try:
                    disposition = meta_data["content-disposition"][1]
                    field_name = disposition["name"].strip()
                except (KeyError, IndexError, AttributeError):
                    continue

                transfer_encoding = meta_data.get("content-transfer-encoding")
                if transfer_encoding is not None:
                    transfer_encoding = transfer_encoding[0].strip()
                field_name = force_str(field_name, encoding, errors="replace")

                if item_type == FIELD:
                    # Avoid reading more than DATA_UPLOAD_MAX_MEMORY_SIZE.
                    if settings.DATA_UPLOAD_MAX_MEMORY_SIZE is not None:
                        read_size = (
                            settings.DATA_UPLOAD_MAX_MEMORY_SIZE - num_bytes_read
                        )

                    # This is a post field, we can just set it in the post
                    if transfer_encoding == "base64":
                        raw_data = field_stream.read(size=read_size)
                        num_bytes_read += len(raw_data)
                        try:
                            data = base64.b64decode(raw_data)
                        except binascii.Error:
                            data = raw_data
                    else:
                        data = field_stream.read(size=read_size)
                        num_bytes_read += len(data)

                    # Add two here to make the check consistent with the
                    # x-www-form-urlencoded check that includes '&='.
                    num_bytes_read += len(field_name) + 2
                    if (
                        settings.DATA_UPLOAD_MAX_MEMORY_SIZE is not None
                        and num_bytes_read > settings.DATA_UPLOAD_MAX_MEMORY_SIZE
                    ):
                        raise RequestDataTooBig(
                            "Request body exceeded "
                            "settings.DATA_UPLOAD_MAX_MEMORY_SIZE."
                        )

                    self._post.appendlist(
                        field_name, force_str(data, encoding, errors="replace")
                    )
                elif item_type == FILE:
                    # Avoid storing more than DATA_UPLOAD_MAX_NUMBER_FILES.
                    num_files += 1
                    if (
                        settings.DATA_UPLOAD_MAX_NUMBER_FILES is not None
                        and num_files > settings.DATA_UPLOAD_MAX_NUMBER_FILES
                    ):
                        raise TooManyFilesSent(
                            "The number of files exceeded "
                            "settings.DATA_UPLOAD_MAX_NUMBER_FILES."
                        )
                    # This is a file, use the handler...
                    file_name = disposition.get("filename")
                    if file_name:
                        file_name = force_str(file_name, encoding, errors="replace")
                        file_name = self.sanitize_file_name(file_name)
                    if not file_name:
                        continue

                    content_type, content_type_extra = meta_data.get(
                        "content-type", ("", {})
                    )
                    content_type = content_type.strip()
                    charset = content_type_extra.get("charset")

                    try:
                        content_length = int(meta_data.get("content-length")[0])
                    except (IndexError, TypeError, ValueError):
                        content_length = None

                    counters = [0] * len(handlers)
                    uploaded_file = False
                    try:
                        for handler in handlers:
                            try:
                                handler.new_file(
                                    field_name,
                                    file_name,
                                    content_type,
                                    content_length,
                                    charset,
                                    content_type_extra,
                                )
                            except StopFutureHandlers:
                                break

                        for chunk in field_stream:
                            if transfer_encoding == "base64":
                                # We only special-case base64 transfer encoding
                                # We should always decode base64 chunks by
                                # multiple of 4, ignoring whitespace.

                                stripped_chunk = b"".join(chunk.split())

                                remaining = len(stripped_chunk) % 4
                                while remaining != 0:
                                    over_chunk = field_stream.read(4 - remaining)
                                    if not over_chunk:
                                        break
                                    stripped_chunk += b"".join(over_chunk.split())
                                    remaining = len(stripped_chunk) % 4

                                try:
                                    chunk = base64.b64decode(stripped_chunk)
                                except Exception as exc:
                                    # Since this is only a chunk, any error is
                                    # an unfixable error.
                                    raise MultiPartParserError(
                                        "Could not decode base64 data."
                                    ) from exc

                            for i, handler in enumerate(handlers):
                                chunk_length = len(chunk)
                                chunk = handler.receive_data_chunk(chunk, counters[i])
                                counters[i] += chunk_length
                                if chunk is None:
                                    # Don't continue if the chunk received by
                                    # the handler is None.
                                    break

                    except SkipFile:
                        self._close_files()
                        # Just use up the rest of this file...
                        exhaust(field_stream)
                    else:
                        # Handle file upload completions on next iteration.
                        old_field_name = field_name
                else:
                    # If this is neither a FIELD nor a FILE, exhaust the field
                    # stream. Note: There could be an error here at some point,
                    # but there will be at least two RAW types (before and
                    # after the other boundaries). This branch is usually not
                    # reached at all, because a missing content-disposition
                    # header will skip the whole boundary.
                    exhaust(field_stream)
        except StopUpload as e:
            self._close_files()
            if not e.connection_reset:
                exhaust(self._input_data)
        else:
            if not uploaded_file:
                for handler in handlers:
                    handler.upload_interrupted()
            # Make sure that the request data is all fed
            exhaust(self._input_data)

        # Signal that the upload has completed.
        # any() shortcircuits if a handler's upload_complete() returns a value.
        any(handler.upload_complete() for handler in handlers)
        self._post._mutable = False
        return self._post, self._files

    def handle_file_complete(self, old_field_name, counters):
        """
        Handle all the signaling that takes place when a file is complete.
        """
        for i, handler in enumerate(self._upload_handlers):
            file_obj = handler.file_complete(counters[i])
            if file_obj:
                # If it returns a file object, then set the files dict.
                self._files.appendlist(
                    force_str(old_field_name, self._encoding, errors="replace"),
                    file_obj,
                )
                break

    def sanitize_file_name(self, file_name):
        """
        Sanitize the filename of an upload.

        Remove all possible path separators, even though that might remove more
        than actually required by the target system. Filenames that could
        potentially cause problems (current/parent dir) are also discarded.

        It should be noted that this function could still return a "filepath"
        like "C:some_file.txt" which is handled later on by the storage layer.
        So while this function does sanitize filenames to some extent, the
        resulting filename should still be considered as untrusted user input.
        """
        file_name = html.unescape(file_name)
        file_name = file_name.rsplit("/")[-1]
        file_name = file_name.rsplit("\\")[-1]
        # Remove non-printable characters.
        file_name = "".join([char for char in file_name if char.isprintable()])

        if file_name in {"", ".", ".."}:
            return None
        return file_name

    IE_sanitize = sanitize_file_name

    def _close_files(self):
        # Free up all file handles.
        # FIXME: this currently assumes that upload handlers store the file as
        # 'file'. We should document that...
        # (Maybe add handler.free_file to complement new_file)
        for handler in self._upload_handlers:
            if hasattr(handler, "file"):
                handler.file.close()


