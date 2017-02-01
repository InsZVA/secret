/**
 * Created by InsZVA on 2017/1/27.
 */

function detectBrowser() {
    var ua = navigator.userAgent.toLowerCase();
    return (ua.match(/rv:([\d.]+)\) like gecko/)) ? "ie" :
        (ua.match(/msie ([\d.]+)/)) ? "ie" :
            (ua.match(/firefox\/([\d.]+)/)) ? "firefox" :
                (ua.match(/chrome\/([\d.]+)/)) ? "chrome" :
                    (ua.match(/opera.([\d.]+)/)) ? "opera" :
                        (ua.match(/version\/([\d.]+).*safari/)) ? "safari" : "unknown";
}

const MIME_VP8 = 'video/webm; codecs="vorbis, vp8"';
const MIME_H264 = 'video/mp4; codecs="avc1.42E01E, mp4a.40.2"';

/**
 * Video class wrapped a video element and buffered video stream from a datastream
 * @param {HTMLVideoElement} v - a html video element
 * @constructor
 */
function Video(v) {
    var browser = detectBrowser();
    // When the browser is chrome or firefox, we use vp8 stream,
    // h264 otherwise.
    this.mime = browser == "chrome" || browser == "firefox" ? MIME_VP8 : MIME_H264;

    // The video element binding to this
    this.v = v;
    if (!window.URL) throw "URL is not supported!";

    // The mediasource
    this.ms = new MediaSource();
    if (!this.ms) throw "The media source is not supported!";
    this.v.src = window.URL.createObjectURL(this.ms);

    // The sourcebuffers of mediasource
    this.sb = [];
}

Video.prototype.fullMime = function (codec) {
    if (codec == "vp9") {
        return "video/webm; codecs=\"vp9\""
    }
    if (codec == "vorbis") {
        return "audio/webm; codecs=\"vorbis\""
    }
};

/**
 * Data stream data coming handler of Video
 * @param {string} codec - the codec of stream
 * @param {Uint8Array} data - the video stream data
 */
Video.prototype.ondata = function (codec, data) {
    if (!this.sb[codec]) {
        //this.ms.addEventListener('sourceopen', function() {
            this.sb[codec] = this.ms.addSourceBuffer(this.fullMime(codec));
            this.sb[codec].appendBuffer(new Uint8Array(data));
       // }.bind(this));
    }

    // TODO: Optimize
    if (this.sb[codec] && !this.sb[codec].updating)
    {
        console.log("play:" + codec);
        this.sb[codec].appendBuffer(new Uint8Array(data));
    }
};
