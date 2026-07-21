(function () {
	'use strict';

	function init() {
		var table = document.querySelector('.table-wrap');
		var toastContainer = document.getElementById('toast-container');
		if (!table || !toastContainer) return;

		var footer = document.getElementById('table-footer');
		var emptyState = document.getElementById('table-empty-state');
		var filterInput = document.getElementById('filter');
		var selectAll = document.getElementById('select-all');
		var selectionCount = document.getElementById('selection-count');
		var bulkDeleteButton = document.getElementById('bulk-delete');
		var baseURL = table.getAttribute('data-base-url') || window.location.origin;
		var editingRow = null;
		var qrPopover = null;
		var qrBackdrop = null;

		function rows() { return Array.prototype.slice.call(table.querySelectorAll('.row:not(.header)')); }
		function visibleRows() { return rows().filter(function (row) { return !row.hidden; }); }
		function selectedRows() { return rows().filter(function (row) { return row.querySelector('.row-checkbox').checked; }); }

		function showToast(message, type) {
			while (toastContainer.children.length >= 3) toastContainer.removeChild(toastContainer.firstChild);
			var toast = document.createElement('div');
			toast.className = 'toast ' + (type || 'info');
			toast.setAttribute('role', type === 'error' ? 'alert' : 'status');
			var text = document.createElement('span');
			text.textContent = message;
			var dismiss = document.createElement('button');
			dismiss.type = 'button';
			dismiss.className = 'toast-dismiss';
			dismiss.setAttribute('aria-label', 'Dismiss notification');
			dismiss.textContent = '×';
			var timer;
			function remove() { clearTimeout(timer); if (toast.parentNode) toast.remove(); }
			function start() { clearTimeout(timer); timer = window.setTimeout(remove, 2500); }
			dismiss.addEventListener('click', remove);
			toast.addEventListener('mouseenter', function () { clearTimeout(timer); });
			toast.addEventListener('mouseleave', start);
			toast.appendChild(text);
			toast.appendChild(dismiss);
			toastContainer.appendChild(toast);
			start();
		}

		function errorMessage(response, fallback) {
			return response.text().then(function (text) {
				try { var body = JSON.parse(text); return body.error || fallback; } catch (_) { return text || fallback; }
			});
		}

		function setButtonBusy(button, busy, label) {
			if (!button) return;
			button.disabled = busy;
			button.setAttribute('aria-busy', String(busy));
			if (label) button.textContent = busy ? 'working…' : label;
		}

		function updateFooter() {
			var all = rows();
			var visible = visibleRows();
			if (footer) footer.textContent = 'showing ' + visible.length + ' of ' + all.length;
			if (emptyState) emptyState.hidden = all.length !== 0;
			table.hidden = all.length === 0;
			if (footer) footer.hidden = all.length === 0;
		}

		function updateSelection() {
			var selected = selectedRows();
			var visible = visibleRows();
			if (selectionCount) selectionCount.textContent = selected.length + ' selected';
			if (bulkDeleteButton) bulkDeleteButton.disabled = selected.length === 0;
			if (selectAll) {
				selectAll.checked = visible.length > 0 && visible.every(function (row) { return row.querySelector('.row-checkbox').checked; });
				selectAll.indeterminate = selected.length > 0 && !selectAll.checked;
			}
		}

		function applyFilter() {
			var query = (filterInput ? filterInput.value : '').trim().toLowerCase();
			rows().forEach(function (row) {
				var haystack = (row.getAttribute('data-code') + ' ' + row.getAttribute('data-url')).toLowerCase();
				row.hidden = query !== '' && haystack.indexOf(query) === -1;
			});
			var existing = table.querySelector('.no-match');
			if (existing) existing.remove();
			if (query && rows().length && visibleRows().length === 0) {
				var noMatch = document.createElement('div');
				noMatch.className = 'no-match';
				noMatch.textContent = 'no matches for “' + query + '”';
				table.appendChild(noMatch);
			}
			updateFooter();
			updateSelection();
		}

		function shortURL(code) { return baseURL.replace(/\/$/, '') + '/' + code; }
		function fallbackCopy(text) {
			var input = document.createElement('textarea');
			input.value = text;
			input.setAttribute('readonly', '');
			input.style.cssText = 'position:fixed;opacity:0;pointer-events:none';
			document.body.appendChild(input);
			input.select();
			var copied = false;
			try { copied = document.execCommand('copy'); } catch (_) {}
			input.remove();
			if (copied) showToast('copied ' + text, 'success'); else showToast('unable to copy URL', 'error');
		}
		function copy(text) {
			if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(text).then(function () { showToast('copied ' + text, 'success'); }, function () { fallbackCopy(text); });
			else fallbackCopy(text);
		}

		function actionButton(action, label, symbol, danger) {
			var button = document.createElement('button');
			button.type = 'button';
			button.className = 'icon-btn' + (danger ? ' danger' : '');
			button.dataset.action = action;
			button.setAttribute('aria-label', label);
			button.title = label;
			button.textContent = symbol;
			return button;
		}
		function restoreRow(row) {
			var code = row.getAttribute('data-code');
			var url = row.getAttribute('data-url');
			var codeCell = row.querySelector('.cell-code');
			var link = document.createElement('a');
			link.href = '/' + encodeURIComponent(code);
			link.target = '_blank'; link.rel = 'noopener noreferrer'; link.textContent = '/' + code;
			codeCell.replaceChildren(link);
			row.querySelector('.cell-url').textContent = url;
			var actions = row.querySelector('.cell-actions');
			actions.replaceChildren(
				actionButton('copy', 'Copy short URL for /' + code, '⊕'), actionButton('qr', 'Show QR code for /' + code, '▦'),
				actionButton('edit', 'Edit /' + code, '✎'), actionButton('delete', 'Delete /' + code, '×', true)
			);
		}
		function exitEdit(row) { row.classList.remove('is-editing'); if (editingRow === row) editingRow = null; restoreRow(row); }
		function enterEdit(row) {
			if (editingRow && editingRow !== row) exitEdit(editingRow);
			closeQR(); editingRow = row; row.classList.add('is-editing');
			var code = row.getAttribute('data-code'); var url = row.getAttribute('data-url');
			var codeInput = document.createElement('input'); codeInput.className = 'edit-code'; codeInput.value = code; codeInput.setAttribute('aria-label', 'Short code');
			var urlInput = document.createElement('input'); urlInput.className = 'edit-url'; urlInput.type = 'url'; urlInput.value = url; urlInput.setAttribute('aria-label', 'Destination URL');
			row.querySelector('.cell-code').replaceChildren(codeInput);
			row.querySelector('.cell-url').replaceChildren(urlInput);
			var actions = row.querySelector('.cell-actions');
			actions.replaceChildren(actionButton('save', 'Save changes', 'save'), actionButton('cancel', 'Discard changes', 'esc'));
			[codeInput, urlInput].forEach(function (input) { input.addEventListener('keydown', function (event) { if (event.key === 'Enter') saveEdit(row); if (event.key === 'Escape') exitEdit(row); }); });
			codeInput.focus(); codeInput.select();
		}
		function saveEdit(row) {
			var code = row.querySelector('.edit-code').value.trim();
			var url = row.querySelector('.edit-url').value.trim();
			var save = row.querySelector('[data-action="save"]');
			setButtonBusy(save, true, 'save');
			fetch('/api/links/' + row.dataset.id, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ short_code: code, original_url: url }) })
				.then(function (response) { if (!response.ok) return errorMessage(response, response.status === 409 ? 'short code already taken' : 'update failed').then(function (message) { throw new Error(message); }); row.dataset.code = code; row.dataset.url = url; exitEdit(row); applyFilter(); showToast('updated /' + code, 'success'); })
				.catch(function (error) { showToast(error.message || 'network error', 'error'); setButtonBusy(save, false, 'save'); });
		}

		function closeQR() { if (qrPopover) qrPopover.remove(); if (qrBackdrop) qrBackdrop.remove(); qrPopover = null; qrBackdrop = null; }
		function openQR(row, button) {
			closeQR();
			qrBackdrop = document.createElement('div'); qrBackdrop.className = 'qr-backdrop'; qrBackdrop.addEventListener('click', closeQR); document.body.appendChild(qrBackdrop);
			qrPopover = document.createElement('div'); qrPopover.className = 'qr-popover'; qrPopover.setAttribute('role', 'dialog'); qrPopover.setAttribute('aria-label', 'QR code for /' + row.dataset.code); qrPopover.textContent = 'loading QR…';
			var rect = button.getBoundingClientRect(); qrPopover.style.top = (rect.bottom + window.scrollY + 6) + 'px'; qrPopover.style.left = Math.min(rect.left + window.scrollX, window.innerWidth - 260) + 'px'; document.body.appendChild(qrPopover);
			fetch('/api/links/' + row.dataset.id + '/qr').then(function (response) { if (!response.ok) return errorMessage(response, 'failed to load QR').then(function (message) { throw new Error(message); }); return response.blob(); })
				.then(function (blob) { if (!qrPopover) return; var image = document.createElement('img'); image.alt = 'QR code for ' + shortURL(row.dataset.code); image.src = URL.createObjectURL(blob); qrPopover.replaceChildren(image); })
				.catch(function (error) { if (qrPopover) qrPopover.textContent = 'failed to load QR'; showToast(error.message || 'failed to load QR', 'error'); });
		}

		table.addEventListener('click', function (event) {
			var button = event.target.closest('[data-action]'); if (!button || button.disabled) return;
			var row = button.closest('.row'); if (!row) return;
			switch (button.dataset.action) {
				case 'copy': copy(shortURL(row.dataset.code)); break;
				case 'qr': openQR(row, button); break;
				case 'edit': enterEdit(row); break;
				case 'save': saveEdit(row); break;
				case 'cancel': exitEdit(row); break;
				case 'delete':
					if (!window.confirm('delete /' + row.dataset.code + '?')) return;
					row.classList.add('is-deleting');
					fetch('/api/links/' + row.dataset.id, { method: 'DELETE' }).then(function (response) { if (!response.ok) return errorMessage(response, 'delete failed').then(function (message) { throw new Error(message); }); row.remove(); applyFilter(); showToast('deleted /' + row.dataset.code, 'success'); }).catch(function (error) { row.classList.remove('is-deleting'); showToast(error.message || 'network error', 'error'); });
			}
		});

		if (filterInput) filterInput.addEventListener('input', applyFilter);
		if (selectAll) selectAll.addEventListener('change', function () { visibleRows().forEach(function (row) { var checkbox = row.querySelector('.row-checkbox'); checkbox.checked = selectAll.checked; row.classList.toggle('is-selected', checkbox.checked); }); updateSelection(); });
		table.addEventListener('change', function (event) { if (!event.target.classList.contains('row-checkbox')) return; var row = event.target.closest('.row'); row.classList.toggle('is-selected', event.target.checked); updateSelection(); });
		if (bulkDeleteButton) bulkDeleteButton.addEventListener('click', function () {
			var selected = selectedRows(); if (!selected.length || !window.confirm('delete ' + selected.length + ' links?')) return;
			setButtonBusy(bulkDeleteButton, true, 'delete selected');
			fetch('/api/links/bulk', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ ids: selected.map(function (row) { return Number(row.dataset.id); }) }) })
				.then(function (response) { if (!response.ok) return errorMessage(response, 'delete failed').then(function (message) { throw new Error(message); }); return response.json(); })
				.then(function (body) {
					if (body.deleted !== selected.length) {
						showToast('deleted ' + body.deleted + ' of ' + selected.length + ' links; refreshing list', 'info');
						window.setTimeout(function () { window.location.reload(); }, 900);
						return;
					}
					selected.forEach(function (row) { row.remove(); });
					showToast('deleted ' + body.deleted + ' links', 'success');
					applyFilter();
				})
				.catch(function (error) { showToast(error.message || 'network error', 'error'); })
				.finally(function () { setButtonBusy(bulkDeleteButton, false, 'delete selected'); updateSelection(); });
		});
		document.addEventListener('keydown', function (event) { if (event.key !== 'Escape') return; if (qrPopover) { closeQR(); return; } if (editingRow) { exitEdit(editingRow); return; } selectedRows().forEach(function (row) { var checkbox = row.querySelector('.row-checkbox'); checkbox.checked = false; row.classList.remove('is-selected'); }); updateSelection(); });
		applyFilter();
	}
	if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init); else init();
})();
