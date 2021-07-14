keyInput = document.getElementById("key");
fileInput = document.getElementById('file');
fileLabel = document.getElementById('fileLabel')
uploadForm = document.getElementById('uploadForm');
statusDiv = document.getElementById('statusDiv');
progress = document.getElementById('progress');
uploadingNow = false;

if (typeof (Storage) !== "undefined") {
    keyInput.value = localStorage.getItem("key");
}

fileLabel.onclick = function (ev) {
    if (uploadingNow) {
        ev.preventDefault();
        return false;
    }
}

fileInput.onchange = function () {
    if (fileInput.files.length) {
        fileLabel.innerHTML = 'Will upload: ' + fileInput.files[0].name + ' (' + fileInput.files[0].size + ')';
    } else {
        fileLabel.innerHTML = 'Browse for a file...';
    }
};

uploadForm.addEventListener('submit', function (ev) {
    if (typeof (Storage) !== "undefined") {
        // save key for later
        localStorage.setItem("key", keyInput.value);
    }

    if (uploadingNow || fileInput.files.length == 0 || fileInput.files[0] == null) {
        ev.preventDefault();
        return false;
    }

    uploadingNow = true;
    statusDiv.innerHTML = "Working...";
    fileLabel.innerHTML = 'Working...';

    var formData = new FormData(uploadForm);
    var req = new XMLHttpRequest();

    fileInput.enabled = false;
    fileInput.value = '';

    req.open("POST", window.location, true);

    req.onload = function () {
        uploadingNow = false;
        fileLabel.innerHTML = 'Browse for a file...';
        if (req.status != 200) {
            statusDiv.innerHTML = "Error " + req.status;
        } else {
            statusDiv.innerHTML = "Ready.";
            statusDiv.innerHTML = 'Uploaded: <a href="' + req.responseText + '">' + req.responseText + '</a>. Ready';
        }
    };

    req.upload.addEventListener('progress', function (e) {
        if (!e.lengthComputable)
            return;

        progress.value = e.loaded * 100 / e.total;
        progress.innerHTML = progress.value.toFixed(0) + '%';
    }, false);

    req.send(formData);
    ev.preventDefault();
}, false);