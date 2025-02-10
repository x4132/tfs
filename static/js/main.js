let _files = [];
let dragCounter = 0;

const handler = {
    get() {
        return Reflect.get(...arguments);
    },

    set(obj, prop, value) {
        const result = Reflect.set(...arguments);
        if (obj.length == 0) {
            document.querySelector("#fileBox").innerHTML = `
<h3 class="text-xl">Drag &amp; Drop or <a href="#" id="selectFile" >Click to Select a File</a></h3>
<p>File uploads up to 200MB are permitted.</p>
`;
        } else {
            let fileDOM = obj.map((file, index) =>
                createFileDOMComponent(file, index)
            );

            document.querySelector("#fileBox").replaceChildren(...fileDOM);
        }

        return result;
    },
};

let files = new Proxy(_files, handler);

window.addEventListener("DOMContentLoaded", () => {
    document.getElementById("year").innerHTML = new Date().getFullYear();
    document
        .querySelector("#fileBox")
        .addEventListener("click", openUploadWindow);
    document
        .querySelector("#authButton")
        .addEventListener("click", setAuthToken);
    document.querySelector("#uploadAll").addEventListener("click", uploadAll);
});

// Drag Functionality
window.addEventListener("dragenter", function dragEnter(evt) {
    evt.preventDefault();
    dragCounter++;
    if (dragCounter === 1) {
        document.querySelector("#dragModal").classList.remove("hidden");
    }
});

window.addEventListener("dragleave", function dragLeave(evt) {
    evt.preventDefault();
    dragCounter--;
    if (dragCounter === 0) {
        document.querySelector("#dragModal").classList.add("hidden");
    }
});

window.addEventListener("dragover", function (evt) {
    evt.preventDefault();
});

window.addEventListener("drop", function drop(evt) {
    evt.preventDefault();
    dragCounter = 0;
    document.querySelector("#dragModal").classList.add("hidden");

    if (evt.dataTransfer.files) {
        let droppedFiles = Array.from(evt.dataTransfer.files);
        droppedFiles.forEach((file) => {
            if (file.size <= 0) {
                return;
            }

            files.push(file);
        });
    }
});

function openUploadWindow() {
    let element = document.createElement("input");
    element.type = "file";

    element.addEventListener("change", function fileInputUpload(evt) {
        if (evt.target.files[0] && evt.target.files[0].size > 0) {
            files.push(evt.target.files[0]);
        }
    });

    element.click();
}

function createFileDOMComponent(file, index) {
    let wrapper = document.createElement("div");
    wrapper.className = "p-2 border rounded-2xl flex items-center";

    let filename = document.createElement("span");
    filename.textContent = file.name;

    let uploadButton = document.createElement("button");
    uploadButton.className = "btn btn-upload btn-upload-individual ml-2";
    uploadButton.textContent = "Upload";

    uploadButton.addEventListener("click", (evt) => {
        evt.target.textContent = "Authenticating...";
        let fileId;
        fetch(
            `/getUploadKey?filename=${encodeURIComponent(file.name)}&filesize=${file.size}`,
            {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                },
            }
        )
            .then((response) => {
                if (response.ok) {
                    evt.target.textContent = "Authenticated.";
                    return response.json();
                } else {
                    evt.target.textContent = "Failed to get Upload Key";
                    throw new Error("Failed to get upload key");
                }
            })
            .then((data) => {
                evt.target.textContent = "Uploading...";
                fileId = data.id;
                fetch(data.url, {
                    method: "PUT",
                    headers: {
                        "Content-Type": file.type,
                    },
                    body: file,
                })
                    .then((uploadResponse) => {
                        if (uploadResponse.ok) {
                        evt.target.innerHTML = `<a class="text-black dark:text-white" href="https:\/\/img.x4132.dev/${fileId}/">https:\/\/img.x4132.dev/${fileId}/</a>`
                            return uploadResponse.json();
                        } else {
                            evt.target.textContent = "Upload Fail";
                            throw new Error("File upload failed");
                        }
                    })
                    .catch((uploadError) => {
                        console.error("Error uploading file:", uploadError);
                    });
            });
    });

    let closeButton = document.createElement("button");
    closeButton.className = "btn btn-close ml-2";
    closeButton.textContent = "X";

    closeButton.addEventListener("click", () => {
        files.splice(index, 1);
    });

    wrapper.appendChild(filename);
    wrapper.appendChild(uploadButton);
    wrapper.appendChild(closeButton);

    wrapper.addEventListener("click", (evt) => evt.stopPropagation());

    return wrapper;
}

function setAuthToken(evt) {
    let token = document.querySelector("#authInput").value;
    if (token === "") {
        alert("no token input");
        return;
    }

    evt.target.textContent = "Authenticating...";

    fetch("/auth", {
        method: "POST",
        headers: {
            "Content-Type": "application/x-www-form-urlencoded",
        },
        body: new URLSearchParams({
            token: token,
        }),
    })
        .then((response) => {
            return response.text();
        })
        .then((responseText) => {
            evt.target.textContent = responseText;
        })
        .catch((err) => {
            evt.target.textContent = err.message;
        });
}

function uploadAll() {
    let uploadBox = document.querySelectorAll(".btn-upload-individual")

    uploadBox.forEach(button => {
        button.click();
    });
}
