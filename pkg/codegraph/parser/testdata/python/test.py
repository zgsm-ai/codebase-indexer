class HttpRequest:
    def parse_file_upload(self, META, post_data):
        """Return a tuple of (POST QueryDict, FILES MultiValueDict)."""
        self.upload_handlers = ImmutableList(
            self.upload_handlers,
            warning=(
                "You cannot alter upload handlers after the upload has been "
                "processed."
            ),
        )
        parser = MultiPartParser(META, post_data, self.upload_handlers, self.encoding)
        return parser.parse()
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