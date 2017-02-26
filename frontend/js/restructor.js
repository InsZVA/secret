/**
 * Created by InsZVA on 2017/2/3.
 */

/**
 * Slice
 * @param {number} cid - chunk id
 * @param {number} sid - slice id
 * @param {number} stotal - slice total
 * @param {Uint8Array} data
 * @constructor
 */
function Slice(cid, sid, stotal, data) {
    this.cid = cid;
    this.sid = sid;
    this.stotal = stotal;
    this.data = data;
}

/**
 * chunk
 * @param {number} id - chunk id
 * @param {Uint8Array} data
 * @param {Uint8Array} raw
 * @constructor
 */
function Chunk(id, data, raw) {
    this.id = id;
    this.data = data;
    this.raw = raw;
}

/**
 * split a chunk to slices
 * @param {number} total
 * @returns {Array}
 */
Chunk.prototype.split = function(total) {
    var data = Array.from(this.data);
    var slicesize = (data.length + total - 1) / total;
    var ret = [];
    for (var i = 0; i < total; i++) {
        ret[i] = new Slice(this.id, i, total, new Uint8Array(
            data.slice(i * slicesize, (i+1) * slicesize)
        ))
    }
    return ret;
};

/**
 * splited chunk
 * @param {number} id - chunk id
 * @param {number} total - total slice
 * @constructor
 */
function SplitedChunk(id, total) {
    this._id = id;
    this._slices = [];
    this._total = total;
    this._full = 0xffffffff >>> (32 - total);
    this._state = 0;
}

/**
 * push a slice to splited chunk
 * @param {Slice} slice
 * @return {Chunk|null}
 */
SplitedChunk.prototype.pushSlice = function(slice) {
    this._slices[slice.sid] = slice.data;
    this._state = this._state | (1 << slice.sid);
    if (this._state == this._full) {
        var data = [];
        for (var i = 0; i < this._total; i++) {
            data = data.concat(Array.from(this._slices[i]));
        }
        return new Chunk(this._id, new Uint8Array(data), null);
    }
    return null;
};

/**
 * Restructor
 * @constructor
 */
function Restructor() {
    this._chunks = [];
    this.onchunk = null;
}

/**
 * push a slice to restructor
 * @param {Slice} slice
 */
Restructor.prototype.pushSlice = function(slice) {
    var tag = "tag" + slice.cid;
    if (!this._chunks[tag]) {
        this._chunks[tag] = new SplitedChunk(slice.cid, slice.stotal);
    }
    var chk;
    if ((chk = this._chunks[tag].pushSlice(slice)) && this.onchunk) {
        this.onchunk(chk);
        this._chunks[tag] = null;
    }
};