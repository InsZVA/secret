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
 * @param {VideoBuffer} vb - a data stream with video
 * @constructor
 */
function Video(v, vb) {
    var browser = detectBrowser();

    // When the browser is chrome or firefox, we use vp8 stream,
    // h264 otherwise.
    this.mime = browser == "chrome" || browser == "firefox" ? MIME_VP8 : MIME_H264;

    // The video element binding to this
    this.v = v;
    if (!window.URL) throw "URL is not supported!";

    // The datastream binding to this
    this.vb = vb;
    this.vb.ondataready = this.ondata.bind(this);

    // The mediasource
    this.ms = new MediaSource();
    if (!this.ms) throw "The media source is not supported!";
    this.v.src = window.URL.createObjectURL(this.ms);

    // The sourcebuffer of mediasource
    this.sb = null;
    this.ms.addEventListener('sourceopen', function() {
        this.sb = this.ms.addSourceBuffer(this.mime);
    }.bind(this));
}

/**
 * Data stream data coming handler of Video
 * @param {Uint8Array} data - the video stream data
 */
Video.prototype.ondata = function (data) {
    if (!this.sb.updating)
        this.sb.appendBuffer(data);
};
