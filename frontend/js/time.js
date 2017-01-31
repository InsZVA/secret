/**
 * Created by InsZVA on 2017/1/28.
 */
const CALIBRATE_INTERVAL = 60 * 1000;

function FixedTime(url) {
    // delta = server ct - local ct
    this.delta = null;
    // calibration times
    this.nCalibration = 0;
    // time calibrate url
    this.url = url;

    setInterval(this.calibrate.bind(this), CALIBRATE_INTERVAL);
}

/**
 * getTimestamp returns a millisecond represent the server time just now
 * @returns {number}
 */
FixedTime.prototype.getTimestamp = function() {
    if (this.delta == null) throw "The time hasn't be calibrated.";
    return this.getLocalTimestamp() + this.delta;
};

FixedTime.prototype.getLocalTimestamp = function() {
    return new Date().getTime();
};

FixedTime.prototype.calibrate = function() {
    var sendTimestamp = this.getLocalTimestamp();

    var oReq = new XMLHttpRequest();
    oReq.addEventListener("load", function() {
        var recvTimestamp = this.getLocalTimestamp();
        var delta = oReq.responseText - (recvTimestamp + sendTimestamp) / 2;
        this.delta = (this.delta * this.nCalibration + delta) / (this.nCalibration + 1);
        this.nCalibration++;
        console.log(this.delta);
    }.bind(this));
    oReq.open("GET", this.url);
    oReq.send();
};

var fixedtime = window.fixedtime = new FixedTime("http://127.0.0.1:8080/time");