import { Writable } from "stream";

export class TempBuffer extends Writable {
    _chunks = [];

    _write(chunk: never, _encoding: BufferEncoding, callback: (error?: Error | null) => void) {
        this._chunks.push(chunk);
        callback();
    }

    getBuffer() {
        return Buffer.concat(this._chunks);
    }
}
