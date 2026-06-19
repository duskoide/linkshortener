(function () {
	var toastContainer = document.getElementById('toast-container');
	var toastTimers = {};

	function showToast(message, type) {
		type = type || 'info';
		while (toastContainer.children.length >= 3) {
			toastContainer.removeChild(toastContainer.firstChild);
		}

		var toast = document.createElement('div');
		toast.className = 'toast ' + type;

		var text = document.createElement('span');
		text.textContent = message;

		var dismiss = document.createElement('span');
		dismiss.className = 'toast-dismiss';
		dismiss.textContent = '×';
		dismiss.onclick = function () { removeToast(toast); };

		toast.appendChild(text);
		toast.appendChild(dismiss);
		toastContainer.appendChild(toast);

		toast.onmouseenter = function () {
			clearTimeout(toastTimers[toast._id]);
		};
		toast.onmouseleave = function () {
			startTimer(toast);
		};

		toast._id = 'toast-' + Date.now() + '-' + Math.random();
		startTimer(toast);
	}

	function startTimer(toast) {
		toastTimers[toast._id] = setTimeout(function () {
			removeToast(toast);
		}, 2500);
	}

	function removeToast(toast) {
		if (toast.parentNode) {
			toast.parentNode.removeChild(toast);
		}
		if (toast._id) {
			clearTimeout(toastTimers[toast._id]);
			delete toastTimers[toast._id];
		}
	}

	function copyToClipboard(text) {
		if (navigator.clipboard && navigator.clipboard.writeText) {
			navigator.clipboard.writeText(text).then(function () {
				showToast('copied ' + text, 'success');
			}).catch(function () {
				fallbackCopy(text);
			});
		} else {
			fallbackCopy(text);
		}
	}

	function fallbackCopy(text) {
		var input = document.createElement('input');
		input.style.position = 'fixed';
		input.style.opacity = '0';
		input.value = text;
		document.body.appendChild(input);
		input.select();
		try {
			document.execCommand('copy');
			showToast('copied ' + text, 'success');
		} catch (e) {
			showToast('failed to copy', 'error');
		}
		document.body.removeChild(input);
	}

	document.addEventListener('DOMContentLoaded', function () {
		var table = document.querySelector('.table-wrap');
		if (!table) return;

		var baseURL = table.getAttribute('data-base-url') || '';
		var location = window.location.origin;

		function buildShortURL(code) {
			if (baseURL) {
				return baseURL + '/' + code;
			}
			return location + '/' + code;
		}

		table.addEventListener('click', function (e) {
			var btn = e.target.closest('[data-action="copy"]');
			if (btn) {
				var row = btn.closest('.row');
				var code = row.getAttribute('data-code');
				var url = buildShortURL(code);
				copyToClipboard(url);
				return;
			}

			var qrBtn = e.target.closest('[data-action="qr"]');
			if (qrBtn) {
				var row = qrBtn.closest('.row');
				openQRPopover(row, qrBtn);
				return;
			}

			var editBtn = e.target.closest('[data-action="edit"]');
			if (editBtn) {
				var row = editBtn.closest('.row');
				enterEditMode(row);
				return;
			}

			var saveBtn = e.target.closest('[data-action="save"]');
			if (saveBtn) {
				saveEdit(saveBtn.closest('.row'));
				return;
			}

			var cancelBtn = e.target.closest('[data-action="cancel"]');
			if (cancelBtn) {
				exitEditMode(cancelBtn.closest('.row'), false);
				return;
			}

			var deleteBtn = e.target.closest('[data-action="delete"]');
			if (deleteBtn) {
				var row = deleteBtn.closest('.row');
				var id = row.getAttribute('data-id');
				var code = row.getAttribute('data-code');

				if (!confirm('delete /' + code + '?')) return;

				row.classList.add('is-deleting');
				fetch('/api/links/' + id, { method: 'DELETE' })
					.then(function (res) {
						if (res.ok) {
							row.remove();
							showToast('deleted /' + code, 'success');
							updateFooter();
						} else {
							row.classList.remove('is-deleting');
							showToast('delete failed', 'error');
						}
					})
					.catch(function () {
						row.classList.remove('is-deleting');
						showToast('network error', 'error');
					});
			}
		});

		function updateFooter() {
			var remaining = table.querySelectorAll('.row:not(.header)').length;
			var footer = document.getElementById('table-footer');
			if (footer) footer.textContent = 'showing ' + remaining + ' of ' + (remaining + table.querySelectorAll('.no-match').length);
		}

		var editingRow = null;

		function enterEditMode(row) {
			if (editingRow && editingRow !== row) {
				exitEditMode(editingRow, false);
			}
			editingRow = row;
			row.classList.add('is-editing');

			var codeCell = row.querySelector('.cell-code');
			var urlCell = row.querySelector('.cell-url');
			var actionsCell = row.querySelector('.cell-actions');

			var code = row.getAttribute('data-code');
			var url = row.getAttribute('data-url');

			codeCell.innerHTML = '<input type="text" class="edit-code" value="' + code + '">';
			urlCell.innerHTML = '<input type="text" class="edit-url" value="' + url + '">';

			actionsCell.innerHTML = '<button class="icon-btn save" data-action="save">save</button><button class="icon-btn" data-action="cancel">esc</button>';

			var codeInput = codeCell.querySelector('input');
			codeInput.focus();
			codeInput.select();

			codeInput.addEventListener('keydown', function (e) {
				if (e.key === 'Enter') saveEdit(row);
				if (e.key === 'Escape') exitEditMode(row, false);
			});
			urlCell.querySelector('input').addEventListener('keydown', function (e) {
				if (e.key === 'Enter') saveEdit(row);
				if (e.key === 'Escape') exitEditMode(row, false);
			});
		}

		function exitEditMode(row, saved) {
			row.classList.remove('is-editing');
			if (editingRow === row) editingRow = null;

			if (!saved) {
				var code = row.getAttribute('data-code');
				var url = row.getAttribute('data-url');
				row.querySelector('.cell-code').innerHTML = '<a href="/' + code + '" target="_blank" rel="noopener noreferrer">/' + code + '</a>';
				row.querySelector('.cell-url').textContent = url;
				restoreActions(row);
			}
		}

		function restoreActions(row) {
			row.querySelector('.cell-actions').innerHTML =
				'<button class="icon-btn" data-action="copy">&#8853;</button>' +
				'<button class="icon-btn" data-action="qr">&#9638;</button>' +
				'<button class="icon-btn" data-action="edit">&#9998;</button>' +
				'<button class="icon-btn danger" data-action="delete">&#10005;</button>';
		}

		function saveEdit(row) {
			var id = row.getAttribute('data-id');
			var codeInput = row.querySelector('.edit-code');
			var urlInput = row.querySelector('.edit-url');
			var newCode = codeInput.value.trim();
			var newUrl = urlInput.value.trim();

			fetch('/api/links/' + id, {
				method: 'PATCH',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ short_code: newCode, original_url: newUrl })
			}).then(function (res) {
				if (res.ok) {
					row.setAttribute('data-code', newCode);
					row.setAttribute('data-url', newUrl);
					exitEditMode(row, true);
					row.querySelector('.cell-code').innerHTML = '<a href="/' + newCode + '" target="_blank" rel="noopener noreferrer">/' + newCode + '</a>';
					row.querySelector('.cell-url').textContent = newUrl;
					restoreActions(row);
					showToast('updated /' + newCode, 'success');
				} else if (res.status === 409) {
					showToast('code already taken', 'error');
				} else {
					return res.json().then(function (data) {
						showToast(data.error || 'update failed', 'error');
					});
				}
			}).catch(function () {
				showToast('network error', 'error');
			});
		}

		var qrPopover = null;
		var qrBackdrop = null;

		function closeQRPopover() {
			if (qrPopover) { qrPopover.remove(); qrPopover = null; }
			if (qrBackdrop) { qrBackdrop.remove(); qrBackdrop = null; }
		}

		function openQRPopover(row, btn) {
			closeQRPopover();
			var id = row.getAttribute('data-id');

			qrBackdrop = document.createElement('div');
			qrBackdrop.className = 'qr-backdrop';
			qrBackdrop.onclick = closeQRPopover;
			document.body.appendChild(qrBackdrop);

			qrPopover = document.createElement('div');
			qrPopover.className = 'qr-popover';
			qrPopover.textContent = 'loading...';

			var rect = btn.getBoundingClientRect();
			qrPopover.style.top = (rect.bottom + window.scrollY + 4) + 'px';
			qrPopover.style.left = (rect.left + window.scrollX) + 'px';
			document.body.appendChild(qrPopover);

			fetch('/api/links/' + id + '/qr')
				.then(function (res) { return res.blob(); })
				.then(function (blob) {
					if (qrPopover) {
						var img = document.createElement('img');
						img.src = URL.createObjectURL(blob);
						qrPopover.textContent = '';
						qrPopover.appendChild(img);
					}
				})
				.catch(function () {
					if (qrPopover) qrPopover.textContent = 'failed to load QR';
				});
		}

		document.addEventListener('keydown', function (e) {
			if (e.key === 'Escape') closeQRPopover();
		});

		var filterInput = document.getElementById('filter');
		if (filterInput) {
			filterInput.addEventListener('input', function () {
				var query = filterInput.value.toLowerCase();
				var rows = table.querySelectorAll('.row:not(.header)');
				var visible = 0;
				rows.forEach(function (row) {
					var code = (row.getAttribute('data-code') || '').toLowerCase();
					var url = (row.getAttribute('data-url') || '').toLowerCase();
					var match = code.indexOf(query) !== -1 || url.indexOf(query) !== -1;
					row.style.display = match ? '' : 'none';
					if (match) visible++;
				});

				var footer = document.getElementById('table-footer');
				if (footer) {
					footer.textContent = 'showing ' + visible + ' of ' + rows.length;
				}

				var noMatch = table.querySelector('.no-match');
				if (noMatch) noMatch.remove();

				if (visible === 0 && query !== '') {
					noMatch = document.createElement('div');
					noMatch.className = 'no-match';
					noMatch.textContent = 'no matches for "' + query + '"';
					table.appendChild(noMatch);
				}
			});
		}

		var selectionCount = document.getElementById('selection-count');
		var bulkDeleteBtn = document.getElementById('bulk-delete');
		var selectAll = document.getElementById('select-all');

		function updateSelection() {
			var checked = table.querySelectorAll('.row-checkbox:checked');
			var count = checked.length;
			if (selectionCount) selectionCount.textContent = count + ' selected';
			if (bulkDeleteBtn) bulkDeleteBtn.disabled = count === 0;
		}

		if (selectAll) {
			selectAll.addEventListener('change', function () {
				var checked = selectAll.checked;
				table.querySelectorAll('.row:not(.header)').forEach(function (row) {
					if (row.style.display !== 'none') {
						row.classList.toggle('is-selected', checked);
						var cb = row.querySelector('.row-checkbox');
						if (cb) cb.checked = checked;
					}
				});
				updateSelection();
			});
		}

		table.addEventListener('change', function (e) {
			if (e.target.classList.contains('row-checkbox')) {
				var row = e.target.closest('.row');
				row.classList.toggle('is-selected', e.target.checked);
				updateSelection();
			}
		});

		if (bulkDeleteBtn) {
			bulkDeleteBtn.addEventListener('click', function () {
				var ids = [];
				table.querySelectorAll('.row-checkbox:checked').forEach(function (cb) {
					ids.push(parseInt(cb.closest('.row').getAttribute('data-id'), 10));
				});
				if (ids.length === 0) return;
				if (!confirm('delete ' + ids.length + ' links?')) return;

				fetch('/api/links/bulk', {
					method: 'DELETE',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ ids: ids })
				}).then(function (res) {
					if (res.ok) {
						ids.forEach(function (id) {
							var row = table.querySelector('.row[data-id="' + id + '"]');
							if (row) row.remove();
						});
						showToast('deleted ' + ids.length + ' links', 'success');
						updateSelection();
						if (selectAll) selectAll.checked = false;
						updateFooter();
					} else {
						showToast('delete failed', 'error');
					}
				}).catch(function () {
					showToast('network error', 'error');
				});
			});
		}

		document.addEventListener('keydown', function (e) {
			if (e.key === 'Escape') {
				table.querySelectorAll('.row-checkbox:checked').forEach(function (cb) {
					cb.checked = false;
					cb.closest('.row').classList.remove('is-selected');
				});
				if (selectAll) selectAll.checked = false;
				updateSelection();
			}
		});
	});
})();