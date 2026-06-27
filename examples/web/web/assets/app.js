function render() {
  document.getElementById('route').textContent = 'Current route: ' + location.pathname;
}

document.addEventListener('click', (e) => {
  const link = e.target.closest('a[data-link]');
  if (!link) return;
  e.preventDefault();
  history.pushState(null, '', link.getAttribute('href'));
  render();
});

window.addEventListener('popstate', render);

// Render the full list of users (name + email) into the <ul>.
document.getElementById('load-users').addEventListener('click', async () => {
  const res = await fetch('/api/v1/users');
  const { users } = await res.json();
  const list = document.getElementById('users');
  list.innerHTML = '';
  for (const u of users) {
    const li = document.createElement('li');
    li.textContent = `#${u.id} — ${u.name} <${u.email}>`;
    list.appendChild(li);
  }
});

// Look up a single user by ID and show the raw JSON response.
document.getElementById('find-user').addEventListener('click', async () => {
  const id = document.getElementById('user-id').value;
  const res = await fetch(`/api/v1/users/${id}`);
  document.getElementById('out').textContent = JSON.stringify(await res.json(), null, 2);
});

render();
