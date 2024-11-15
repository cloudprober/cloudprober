function markConfigDocLinkActive() {
  const configDocLinks = document.getElementById("config-doc-links");
  for (const a of configDocLinks.getElementsByTagName("a")) {
    if (a.href.replace(/\/$/, "") == window.location.href.replace(/\/$/, "")) {
      a.classList.add("config-link-active");
    }
  }
}

/**
 * @param {string} lang
 * @returns {void}
 * */
function setConfigLang(lang) {
  if (!lang) {
    lang = localStorage["preferred-language"];
  }
  if (!lang) {
    lang = "textpb";
  }

  const els = document.getElementsByClassName("select-content-lang");
  for (const e of els) {
    if (e.id == "content-lang-" + lang) {
      e.style.display = "block";
      for (const h3 of e.getElementsByTagName("h3")) {
        h3.id = h3.id.replace(":disabled", "");
      }
    } else {
      e.style.display = "none";
      for (const h3 of e.getElementsByTagName("h3")) {
        h3.id = h3.id + ":disabled";
      }
    }
  }
  localStorage["preferred-language"] = lang;

  const setLangButtons = document.getElementsByClassName("set-lang");
  for (const b of setLangButtons) {
    b.addEventListener("click", async function () {
      setConfigLang(b.id.replace("set-lang-", ""));
    });

    if (b.id == "set-lang-" + lang) {
      b.classList.add("btn-primary");
    } else {
      b.classList.remove("btn-primary");
    }
  }
}

// subPackages arranges messages into subpackages/**
/**
 * @param {string[]} msgs
 * @returns {Object<string, string[]>}
 */
function parseSubPkgs(msgs) {
  const subPackages = new Object();
  for (const msg of msgs) {
    const parts = msg.split(".", 4);
    let sp = parts.slice(0, 2).join(".");
    if (parts.length > 3) {
      sp = parts.slice(0, 3).join(".");
    }
    if (!subPackages[sp]) {
      subPackages[sp] = [];
    }
    subPackages[sp].push(msg);
  }
  return subPackages;
}

/**
 * @param {string[]} msgs
 * @param {HTMLElement} ul
 * @param {string} prefix
 * @param {boolean} addAHandler
 * @returns {void}
 * */
function addFlatTOC(msgs, ul, prefix, addAHandler) {
  for (const msg of msgs) {
    const li = document.createElement("li");
    li.style.whiteSpace = "nowrap";
    const a = document.createElement("a");
    li.appendChild(a);
    a.href = "#" + msg.replaceAll(".", "_");
    a.innerText = msg.replace(prefix, "");
    a.style.fontSize = "0.90rem";
    a.style.fontWeight = 400;
    if (addAHandler) {
      a.addEventListener("click", function (e) {
        e.stopPropagation();
        ul.style.display = "block";
      });
    }
    ul.appendChild(li);
  }
}

/**
 * @param {string[]} names
 * @param {HTMLElement} tocUL
 * @returns {void}
 */
function buildTOCFromNames(names, tocUL) {
  const subPackages = parseSubPkgs(names);
  for (const name in subPackages) {
    if (name.split(".").length == 2) {
      addFlatTOC(subPackages[name], tocUL, "", false);
      continue;
    }
    const li = document.createElement("li");
    const buttonTmpl = document.createElement("template");
    buttonTmpl.innerHTML =
      '<button class="btn btn-toggle align-items-center rounded collapsed" style="font-size:0.90rem" data-bs-toggle="collapse">' +
      name +
      "</button>";
    li.appendChild(buttonTmpl.content.firstChild);
    li.style.whiteSpace = "nowrap";
    tocUL.appendChild(li);

    const ul = document.createElement("ul");
    li.classList.add("btn-toggle-nav");
    li.addEventListener("click", function () {
      if (ul.style.display == "none") {
        ul.style.display = "block";
      } else {
        ul.style.display = "none";
      }
    });
    const h = window.location.hash.replace("#", "");
    if (h?.startsWith(name)) {
      ul.style.display = "block";
    } else {
      ul.style.display = "none";
    }
    li.appendChild(ul);
    addFlatTOC(subPackages[name], ul, name, true);
  }
}

function addTOC() {
  const tocDivs = document.getElementsByClassName("docs-toc");
  if (!tocDivs || tocDivs.length == 0) {
    console.log("TOC div not found");
    return;
  }
  const tocTmpl = document.createElement("template");
  tocTmpl.innerHTML =
    '<div class="page-links d-none d-xl-block"><nav id="TableOfContents"><ul></ul></nav></div>';
  tocDivs[0].appendChild(tocTmpl.content.firstChild);
  const tocUL = document
    .getElementById("TableOfContents")
    .getElementsByTagName("ul")[0];

  for (const el of document.getElementsByClassName("select-content-lang")) {
    if (el.style.display == "none") continue;

    const h3s = el.getElementsByTagName("h3");
    if (h3s.length == 0) continue;

    let names = [];
    for (const h3 of h3s) {
      names.push(h3.innerText.trim());
    }

    buildTOCFromNames(names, tocUL);
  }
}

function configVersionSelector() {
  var currentVersion = "latest";
  var configName = "overview";
  const configDocsPath = (version, configName) => `/docs/config/${version}/${configName}/`;
  const url = new URL(window.location.href);
  const urlParts = url.pathname.split("/");
  // URL with version will have 6 parts.
  if (urlParts.length == 6) {
    configName = urlParts[4];
    currentVersion = urlParts[3];
  } else if (urlParts.length == 5) {
    configName = urlParts[3];
  }
  
  const versionSelector = document.getElementById("config-version-selector");
  if (versionSelector) {
    versionSelector.value = currentVersion;
    versionSelector.addEventListener("change", function () {
      const selectedVersion = versionSelector.value;
      url.pathname = configDocsPath(selectedVersion, configName);
      window.location.href = url.toString();
    });
  }
}

document.addEventListener("DOMContentLoaded", function () {
  setConfigLang();
  configVersionSelector();
  if (window.location.hash) {
    const e = document.querySelector(window.location.hash);
    if (e) {
      e.scrollIntoView();
    }
  }
  markConfigDocLinkActive();
  addTOC();
});
