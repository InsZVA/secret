/**
 * Created by InsZVA on 2017/2/3.
 */

function _bigendian() {}
window.bigendian = new _bigendian();

/**
 * read a uint32
 * @param {ArrayBuffer|Uint8Array} data
 * @returns {number}
 */
_bigendian.prototype.readUint32 = function(data) {
    if (data.length < 4) throw "Bigendian: data overflow";
    var ret = 0;
    for (var i = 0; i < 4; i++) {
        ret <<= 8;
        ret |= data[i];
    }
    return ret;
};

/**
 * read a uint64
 * @param {Uint8Array} data
 * @returns {number}
 */
_bigendian.prototype.readUint64 = function(data) {
    if (data.length < 8) throw "Bigendian: data overflow";
    var ret = 0;
    for (var i = 0; i < 8; i++) {
        ret *= 256;
        ret += data[i];
    }
    return ret;
};

/**
 * read a string
 * @param {Uint8Array} data
 * @param {number} length
 * @returns {string}
 */
_bigendian.prototype.readString = function(data, length) {
    if (data.length < length) throw "Bigendian: data overflow";
    return new TextDecoder("utf-8").decode(data.slice(0, length));
};