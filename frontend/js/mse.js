/**
 * Created by InsZVA on 2017/2/3.
 */

/**
 * init message of a track
 * @param {Uint8Array} raw
 * @constructor
 */
function InitMsg(raw) {
    var offset = bigendian.readUint32(raw);
    var codec = bigendian.readString(raw.slice(4), offset - 4);
    this.codec = this.fullmimie(codec);
    this.data = new Uint8Array(raw.slice(offset));
}

/**
 * return full mime of short mime
 * @param short
 * @returns {string}
 */
InitMsg.prototype.fullmimie = function(short) {
    switch (short) {
        case "vp9":
            return "video/webm; codecs=\"vp9\"";
        case "vorbis":
            return "audio/webm; codecs=\"vorbis\"";
    }
    return "unsupport";
};

/**
 * MediaSourceExtension
 * @param {HTMLVideoElement} v
 * @param {InitMsg} vinit
 * @param {InitMsg} ainit
 * @constructor
 */
function MSE(v, vinit, ainit) {
    if (!window.URL) throw "This browser dosn't support URL";
    this._ms = new MediaSource();
    this._v = v;
    this._sb = [];
    this._sbqueue = [];
    this._vinit = vinit;
    this._ainit = ainit;
    this._ms.addEventListener('sourceopen', this.init.bind(this));
    this._v.src = URL.createObjectURL(this._ms);
}

/**
 * init
 */
MSE.prototype.init = function() {
    this._sb[0] = this._ms.addSourceBuffer(this._vinit.codec);
    this._sb[1] = this._ms.addSourceBuffer(this._ainit.codec);
    this._sbqueue[0] = [];
    this._sbqueue[1] = [];

    this._sb[0].appendBuffer(this._vinit.data);
    this._sb[1].appendBuffer(this._ainit.data);
    this._sb[0].addEventListener('updateend', this.updatelistener.call(this, 0));
    this._sb[1].addEventListener('updateend', this.updatelistener.call(this, 1));
};

/**
 * create a update listener
 * @param {number} index
 * @returns {Function}
 */
MSE.prototype.updatelistener = function(index) {
    return function() {
        if (this._sbqueue[index].length > 0) {
            this._sb[index].appendBuffer(this._sbqueue[index][0]);
            this._sbqueue[index] = this._sbqueue[index].slice(1);
        }

        if (this._sb[index].buffered.length > 0 &&
            this._sb[index].buffered.start(0) > this._v.currentTime)
            this._v.currentTime = this._sb[index].buffered.start(0) + 0.05;
    }.bind(this);
};

/**
 * sync a chunk to mse
 * @param {number} index
 * @param {Chunk} chunk
 */
MSE.prototype.syncChunk = function(index, chunk) {
    index = parseInt(index);
    if (this._sb[index].updating)
        this._sbqueue.push(chunk.data);
    else
        this._sb[index].appendBuffer(chunk.data);

    if (this._sb[index].buffered.length > 0 &&
        this._sb[index].buffered.end(0) - this._sb[index].buffered.start(0) > 120)
        this._sb[index].remove(this._sb[index].buffered.start(0),
            this._sb[index].buffered.start(0) + 60);
};