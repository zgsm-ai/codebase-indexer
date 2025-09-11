import * as documents from "./_namespaces/documents.js";
import { Compiler } from "./_namespaces/Harness.js";
import * as ts from "./_namespaces/ts.js";
import * as Utils from "./_namespaces/Utils.js";

interface SourceMapSpanWithDecodeErrors {
    sourceMapSpan: ts.Mapping;
    decodeErrors: string[] | undefined;
}

namespace SourceMapDecoder {
    let sourceMapMappings: string;
    let decodingIndex: number;
    let mappings: ts.MappingsDecoder | undefined;

    export interface DecodedMapping {
        sourceMapSpan: ts.Mapping;
        error?: string;
    }

    export function initializeSourceMapDecoding(sourceMap: ts.RawSourceMap) {
        decodingIndex = 0;
        sourceMapMappings = sourceMap.mappings;
        mappings = ts.decodeMappings(sourceMap.mappings);
    }

    export function decodeNextEncodedSourceMapSpan(): DecodedMapping {
        if (!mappings) return ts.Debug.fail("not initialized");
        const result = mappings.next();
        if (result.done) return { error: mappings.error || "No encoded entry found", sourceMapSpan: mappings.state };
        return { sourceMapSpan: result.value };
    }

    export function hasCompletedDecoding() {
        if (!mappings) return ts.Debug.fail("not initialized");
        return mappings.pos === sourceMapMappings.length;
    }

    export function getRemainingDecodeString() {
        return sourceMapMappings.substr(decodingIndex);
    }
}

namespace SourceMapSpanWriter {
    let sourceMapRecorder: Compiler.WriterAggregator;
    let sourceMapSources: string[];
    let sourceMapNames: string[] | null | undefined; // eslint-disable-line no-restricted-syntax

    let jsFile: documents.TextDocument;
    let jsLineMap: readonly number[];
    let tsCode: string;
    let tsLineMap: number[];

    let spansOnSingleLine: SourceMapSpanWithDecodeErrors[];
    let prevWrittenSourcePos: number;
    let nextJsLineToWrite: number;
    let spanMarkerContinues: boolean;

    export function initializeSourceMapSpanWriter(sourceMapRecordWriter: Compiler.WriterAggregator, sourceMap: ts.RawSourceMap, currentJsFile: documents.TextDocument) {
        sourceMapRecorder = sourceMapRecordWriter;
        sourceMapSources = sourceMap.sources;
        sourceMapNames = sourceMap.names;

        jsFile = currentJsFile;
        jsLineMap = jsFile.lineStarts;

        spansOnSingleLine = [];
        prevWrittenSourcePos = 0;
        nextJsLineToWrite = 0;
        spanMarkerContinues = false;

        SourceMapDecoder.initializeSourceMapDecoding(sourceMap);
        sourceMapRecorder.WriteLine("===================================================================");
        sourceMapRecorder.WriteLine("JsFile: " + sourceMap.file);
        sourceMapRecorder.WriteLine("mapUrl: " + ts.tryGetSourceMappingURL(ts.getLineInfo(jsFile.text, jsLineMap)));
        sourceMapRecorder.WriteLine("sourceRoot: " + sourceMap.sourceRoot);
        sourceMapRecorder.WriteLine("sources: " + sourceMap.sources);
        if (sourceMap.sourcesContent) {
            sourceMapRecorder.WriteLine("sourcesContent: " + JSON.stringify(sourceMap.sourcesContent));
        }
        sourceMapRecorder.WriteLine("===================================================================");
    }

    function getSourceMapSpanString(mapEntry: ts.Mapping, getAbsentNameIndex?: boolean) {
        let mapString = "Emitted(" + (mapEntry.generatedLine + 1) + ", " + (mapEntry.generatedCharacter + 1) + ")";
        if (ts.isSourceMapping(mapEntry)) {
            mapString += " Source(" + (mapEntry.sourceLine + 1) + ", " + (mapEntry.sourceCharacter + 1) + ") + SourceIndex(" + mapEntry.sourceIndex + ")";
            if (mapEntry.nameIndex! >= 0 && mapEntry.nameIndex! < sourceMapNames!.length) {
                mapString += " name (" + sourceMapNames![mapEntry.nameIndex!] + ")";
            }
            else {
                if ((mapEntry.nameIndex && mapEntry.nameIndex !== -1) || getAbsentNameIndex) {
                    mapString += " nameIndex (" + mapEntry.nameIndex + ")";
                }
            }
        }

        return mapString;
    }

    export function recordSourceMapSpan(sourceMapSpan: ts.Mapping) {
        // verify the decoded span is same as the new span
        const decodeResult = SourceMapDecoder.decodeNextEncodedSourceMapSpan();
        let decodeErrors: string[] | undefined;
        if (typeof decodeResult.error === "string" || !ts.sameMapping(decodeResult.sourceMapSpan, sourceMapSpan)) {
            if (decodeResult.error) {
                decodeErrors = ["!!^^ !!^^ There was decoding error in the sourcemap at this location: " + decodeResult.error];
            }
            else {
                decodeErrors = ["!!^^ !!^^ The decoded span from sourcemap's mapping entry does not match what was encoded for this span:"];
            }
            decodeErrors.push("!!^^ !!^^ Decoded span from sourcemap's mappings entry: " + getSourceMapSpanString(decodeResult.sourceMapSpan, /*getAbsentNameIndex*/ true) + " Span encoded by the emitter:" + getSourceMapSpanString(sourceMapSpan, /*getAbsentNameIndex*/ true));
        }

        if (spansOnSingleLine.length && spansOnSingleLine[0].sourceMapSpan.generatedLine !== sourceMapSpan.generatedLine) {
            // On different line from the one that we have been recording till now,
            writeRecordedSpans();
            spansOnSingleLine = [];
        }
        spansOnSingleLine.push({ sourceMapSpan, decodeErrors });
    }
}