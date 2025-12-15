const storageKey = "pbCategories";
let cachedPseudo = null;

async function fetchPseudo() {
    if (cachedPseudo) {
        return cachedPseudo;
    }
    try {
        const response = await fetch("/api/user");
        if (!response.ok) {
            return "";
        }
        const data = await response.json();
        if (data && data.pseudo) {
            cachedPseudo = data.pseudo;
            return cachedPseudo;
        }
    } catch (err) {
        console.error("PetitBac: impossible de recuperer l'utilisateur:", err);
    }
    return "";
}

function getStoredCategories() {
    try {
        const raw = sessionStorage.getItem(storageKey);
        if (!raw) {
            return [];
        }
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
            return parsed;
        }
    } catch (e) {
        console.warn("PetitBac: categories invalides dans le storage");
    }
    return [];
}

function saveCategories(categories) {
    sessionStorage.setItem(storageKey, JSON.stringify(categories));
}

function setupCategoriesStep() {
    const form = document.getElementById("categoriesForm");
    if (!form) {
        return;
    }
    form.addEventListener("submit", function (event) {
        event.preventDefault();
        const selected = Array.from(form.querySelectorAll("input[name='categories']:checked"))
            .map(input => input.value.trim())
            .filter(Boolean);
        const custom = form.querySelector("#custom-category");
        if (custom && custom.value.trim() !== "") {
            selected.push(custom.value.trim());
        }

        const unique = Array.from(new Set(selected));
        if (unique.length === 0) {
            alert("Merci de sélectionner au moins une catégorie.");
            return;
        }
        saveCategories(unique);
        window.location.href = "/PetitBac/create/time";
    });
}

function populateCategorySummary(list) {
    const summary = document.getElementById("categorySummary");
    if (!summary) {
        return;
    }
    summary.innerHTML = "";
    if (!list.length) {
        const li = document.createElement("li");
        li.textContent = "Aucune catégorie sélectionnée. Retour à l'étape 1.";
        summary.appendChild(li);
        return;
    }
    list.forEach(cat => {
        const li = document.createElement("li");
        li.textContent = cat;
        summary.appendChild(li);
    });
}

function setupTimeStep() {
    const form = document.getElementById("timeForm");
    if (!form) {
        return;
    }
    const categories = getStoredCategories();
    populateCategorySummary(categories);
    if (!categories.length) {
        const error = document.getElementById("timeError");
        if (error) {
            error.textContent = "Aucune catégorie trouvée. Reprends l'étape 1.";
        }
    }

    form.addEventListener("submit", async function (event) {
        event.preventDefault();
        const error = document.getElementById("timeError");
        if (error) {
            error.textContent = "";
        }
        if (!categories.length) {
            if (error) {
                error.textContent = "Merci de sélectionner des catégories avant de créer un salon.";
            }
            return;
        }
        const duration = parseInt(document.getElementById("duration").value, 10);
        const rounds = parseInt(document.getElementById("rounds").value, 10);
        const host = await fetchPseudo();

        try {
            const response = await fetch("/PetitBac/salons", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({
                    categories: categories,
                    temps: duration,
                    manches: rounds,
                    host: host
                })
            });
            if (!response.ok) {
                throw new Error("Creation echouee");
            }
            const data = await response.json();
            sessionStorage.removeItem(storageKey);
            window.location.href = "/PetitBac/wait?room=" + encodeURIComponent(data.code);
        } catch (err) {
            console.error(err);
            if (error) {
                error.textContent = "Impossible de créer le salon. Réessaie.";
            }
        }
    });
}

function setupJoinStep() {
    const form = document.getElementById("joinRoomForm");
    if (!form) {
        return;
    }
    form.addEventListener("submit", async function (event) {
        event.preventDefault();
        const codeInput = document.getElementById("room");
        const message = document.getElementById("joinError");
        if (message) {
            message.textContent = "";
        }
        const code = (codeInput.value || "").trim().toUpperCase();
        if (!code) {
            if (message) {
                message.textContent = "Merci de saisir un code.";
            }
            return;
        }
        try {
            const response = await fetch("/PetitBac/salons/join", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({code})
            });
            if (!response.ok) {
                const text = await response.text();
                throw new Error(text || "Code invalide");
            }
            window.location.href = "/PetitBac/wait?room=" + encodeURIComponent(code);
        } catch (err) {
            console.error(err);
            if (message) {
                message.textContent = "Impossible de rejoindre ce salon.";
            }
        }
    });
}

document.addEventListener("DOMContentLoaded", function () {
    setupCategoriesStep();
    setupTimeStep();
    setupJoinStep();
});
