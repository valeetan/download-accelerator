const form = document.querySelector('form');
const result = document.getElementById('result');
const tpl = document.getElementById('tpl-result');
form.addEventListener('submit', async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  const resp = await fetch('/', { method: 'POST', body: fd });
  const data = await resp.json();
  if (!data || !data.link) { result.textContent = '生成失败'; return }
  const node = tpl.content.cloneNode(true);
  node.querySelector('[data-k="filename"]').textContent = data.meta?.filename || '';
  node.querySelector('[data-k="size"]').textContent = formatSize(data.meta?.size);
  node.getElementById('btnDownload').href = data.link;
  node.getElementById('btnCopy').addEventListener('click', async () => {
    await navigator.clipboard.writeText(data.link);
  });
  result.innerHTML = '';
  result.appendChild(node);
});

function formatSize(n){
  if (!n || n < 0) return '未知';
  const u=['B','KB','MB','GB','TB'];
  let i=0, x=n;
  while(x>=1024 && i<u.length-1){ x/=1024; i++; }
  return x.toFixed(1)+' '+u[i];
}

