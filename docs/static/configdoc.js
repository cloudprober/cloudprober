const markConfigDocLinkActive = async function () {
  const configDocLinks = document.getElementById("config-doc-links");
  for (const a of configDocLinks.getElementsByTagName("a")) {
    if (a.href.replace(/\/$/, "") == window.location.href.replace(/\/$/, "")) {
      a.classList.add("config-link-active");
    }
  }
};

const setConfigLang = function (lang) {
  if (!lang) {
    lang = localStorage["preferred-language"];
  }
  if (!lang) {
    lang = "yaml";
  }

  const els = document.getElementsByClassName("select-content-lang");
  for (const e of els) {
    if (e.id == "content-lang-" + lang) {
      e.style.display = "block";
    } else {
      e.style.display = "none";
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
};

// subPackages arranges messages into subpackages
const subPackages = function (msgs) {
  const subPackages = {};
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
};

const addFlatTOC = function (msgs, ul, name, addAHandler) {
  for (const msg of msgs) {
    const li = document.createElement("li");
    ul.appendChild(li);
    li.style.whiteSpace = "nowrap";
    const a = document.createElement("a");
    li.appendChild(a);

    a.href = "#" + msg;
    a.innerText = msg.replace(name, "");
    a.style.fontSize = "0.90rem";
    a.style.fontWeight = 400;
    if (addAHandler) {
      a.addEventListener("click", function (e) {
        e.stopPropagation();
        ul.style.display = "block";
      });
    }
  }
};

const addTOC = function () {
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

  const els = document.getElementsByClassName("select-content-lang");
  for (const el of els) {
    if (el.style.display == "none") continue;

    const h3s = el.getElementsByTagName("h3");
    if (h3s.length == 0) {
      continue;
    }

    let names = [];
    for (const h3 of h3s) {
      names.push(h3.innerText);
    }

    const subpkgs = subPackages(names);
    const subPackageNames = [];
    for (const sp in subpkgs) {
      subPackageNames.push(sp);
    }
    subPackageNames.sort();

    for (const name of subPackageNames) {
      if (name.split(".").length == 2) {
        addFlatTOC(subpkgs[name], tocUL, "", false);
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
      if (h && h.startsWith(name)) {
        ul.style.display = "block";
      } else {
        ul.style.display = "none";
      }
      li.appendChild(ul);
      addFlatTOC(subpkgs[name], ul, name, true);
    }
  }
};

document.addEventListener("DOMContentLoaded", async function () {
  setConfigLang();
  markConfigDocLinkActive();
  for (const el of document.getElementsByClassName("protodoc")) {
    el.style.borderLeft = "3px solid #e76f51";
  }
  addTOC();
});
