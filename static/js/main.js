function CreateRoom() {
  $('.fullscreen.modal').modal('show');
  if (App.token.length <= 0) {
    GetToken();
  } else {
    SetFormValueToken(App.token);
  }
}
function AddMessage() {
  $('.fullscreen.modal').modal('show');
  $('.ui.radio.checkbox').checkbox();
  if (App.token.length <= 0) {
    GetToken();
  } else {
    SetFormValueToken(App.token);
  }
}
function CloseModal() {
  $('.fullscreen.modal').modal('hide');
}
function ChangeIcon(icon_id) {
  if (icon_id == 1) {
    $('#iconimg').attr('src', '{{template "icon1.jpg" .}}');
  } else {
    $('#iconimg').attr('src', '{{template "icon2.jpg" .}}');
  }
}
function SubmitForm(elm) {
  $(elm).addClass("disabled");
  var data = $(elm).closest('form').serializeArray();
  data = parseJson(data);
  $.ajax({
    type:          'POST',
    url:           App.url,
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data)
  })
  .always(function() {
    window.setTimeout(() => location.reload(true), 1000);
  });
}
function Report(elm) {
  if ($(elm).children('a').text() == 'Reported') {
    return;
  }
  $(elm).addClass('disabled');
  var data = $(elm).closest('form').serializeArray();
  data = parseJson(data);
  $.ajax({
    type:          'POST',
    url:           App.url,
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data)
  })
  .always(function() {
    $(elm).children('a').text('Reported');
  });
}
function GetToken() {
  const data = {action: 'puttoken'};
  $.ajax({
    type:          'POST',
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data),
    url:           App.url
  })
  .done(function(res) {
    App.token = res.token;
    SetFormValueToken(App.token);
  })
  .fail(function(e) {
    console.log(e);
  });
}
function SetFormValueToken(token) {
  $('#token').attr('value', token);
}
var parseJson = function(data) {
  var res = {};
  for (i = 0; i < data.length; i++) {
    res[data[i].name] = data[i].value
  }
  return res;
};
var App = { token: '', url: location.origin + {{ .ApiPath }} };
