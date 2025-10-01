const form = document.querySelector('form');
const result = document.getElementById('result');
form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  const resp = await fetch('/', { method: 'POST', body: fd });
  const text = await resp.text();
  result.innerHTML = '<h3>下载链接</h3><pre>'+text+'</pre>';
});

