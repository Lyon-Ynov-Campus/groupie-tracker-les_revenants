async function checkAuthStatus() {
    try {
        const response = await fetch('/api/user');
        const data = await response.json();
        
        const navLinks = document.getElementById('navLinks');
        
        if (data.authenticated) {
            navLinks.innerHTML = `
                <span class="user-info">Salut, ${data.pseudo} 🎀</span>
                <a href="/logout" class="btn-secondary">Déconnexion</a>
            `;
        }
    } catch (error) {
        console.error('Erreur lors de la vérification de l\'authentification:', error);
    }
}

checkAuthStatus();
