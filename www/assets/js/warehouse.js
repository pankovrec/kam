$(document).ready(function () {
    // Обработчики для товаров
    $(document)
        // Добавление товара
        .on('submit', '#addItemModal form', function (e) {
            e.preventDefault();
            const $form = $(this);
            submitItemForm($form, function () {
                $('#addItemModal').modal('hide');
                $form[0].reset();
                refreshItemsTable();
                showAlert('Товар успешно добавлен!', 'success');
            });
        })

        // Редактирование товара
        .on('submit', '#editItemModal form', function (e) {
            e.preventDefault();
            const $form = $(this);
            submitItemForm($form, function () {
                $('#editItemModal').modal('hide');
                refreshItemsTable();
                showAlert('Товар успешно обновлен!', 'success');
            });
        })

        // Удаление товара
        .on('click', '.delete-item', function (e) {
            e.preventDefault();
            const itemId = $(this).data('id');
            confirmAction('Удалить этот товар?', function () {
                $.ajax({
                    url: '/deleteItem?id=' + itemId,
                    method: 'GET',
                    success: function () {
                        $('tr[data-id="' + itemId + '"]').remove();
                        showAlert('Товар удален', 'success');
                    },
                    error: showError
                });
            });
        })

        // Открытие формы редактирования товара
        .on('click', '.edit-item', function (e) {
            e.preventDefault();
            const itemId = $(this).data('id');
            loadItemData(itemId);
        });

    // Обработчики для категорий
    $(document)
        // Добавление категории
        .on('submit', '#addCategoryModal form', function (e) {
            e.preventDefault();
            const $form = $(this);
            submitCategoryForm($form, function () {
                $('#addCategoryModal').modal('hide');
                $form[0].reset();
                refreshCategoriesTable();
                showAlert('Категория добавлена!', 'success');
            });
        })

        // Редактирование категории
        .on('submit', '#editCategoryModal form', function (e) {
            e.preventDefault();
            const $form = $(this);
            submitCategoryForm($form, function () {
                $('#editCategoryModal').modal('hide');
                refreshCategoriesTable();
                showAlert('Категория обновлена!', 'success');
            });
        })

        // Удаление категории
        .on('click', '.delete-category', function (e) {
            e.preventDefault();
            const url = $(this).attr('href');
            confirmAction('Удалить эту категорию?', function () {
                $.ajax({
                    url: url,
                    method: 'GET',
                    success: function () {
                        refreshCategoriesTable();
                        showAlert('Категория удалена', 'success');
                    },
                    error: showError
                });
            });
        })

        // Открытие формы редактирования категории
        .on('click', '.edit-category', function (e) {
            e.preventDefault();
            const $row = $(this).closest('tr');
            const categoryId = $(this).data('id');
            const categoryName = $row.find('td:first').text();

            $('#editCategoryId').val(categoryId);
            $('#editCategoryName').val(categoryName);
            $('#editCategoryModal').modal('show');
        });

    // Вспомогательные функции
    function submitItemForm($form, successCallback) {
        $.ajax({
            url: $form.attr('action'),
            method: $form.attr('method'),
            data: $form.serialize(),
            success: successCallback,
            error: showError
        });
    }

    function submitCategoryForm($form, successCallback) {
        $.ajax({
            url: $form.attr('action'),
            method: $form.attr('method'),
            data: $form.serialize(),
            success: successCallback,
            error: showError
        });
    }

    function loadItemData(itemId) {
        $.get('/getItem?id=' + itemId)
            .done(function (item) {
                $('#editItemId').val(item.id);
                $('#editItemName').val(item.name);
                $('#editItemQty').val(item.qty);
                $('#editItemComments').val(item.comments);
                $('#editItemCategory').val(item.categorie_id);
                $('#editItemModal').modal('show');
            })
            .fail(showError);
    }

    function refreshItemsTable() {
        $.get(window.location.href, function (data) {
            const newTable = $(data).find('#items table').html();
            $('#items table').html(newTable);
        });
    }

    function refreshCategoriesTable() {
        $.get(window.location.href, function (data) {
            const newTable = $(data).find('#categories table').html();
            $('#categories table').html(newTable);
        });
    }

    function confirmAction(message, callback) {
        if (confirm(message)) {
            callback();
        }
    }

    function showError(xhr) {
        console.error('Error:', xhr.responseText);
        showAlert('Ошибка: ' + xhr.responseText, 'danger');
    }



    function showAlert(message, type) {
        // Реализуйте красивый alert или используйте toast-уведомления
        alert(message); // Временное решение
    }
});