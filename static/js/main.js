$(document).ready(function() {
  var
    $headers     = $('body > div > h1'),
    $header      = $headers.first(),
    ignoreScroll = false,
    timer;

  $(window)
    .on('resize', function() {
      clearTimeout(timer);
      $headers.visibility('disable callbacks');

      $(document).scrollTop( $header.offset().top );

      timer = setTimeout(function() {
        $headers.visibility('enable callbacks');
      }, 500);
    });
  $headers
    .visibility({
      once: false,
      checkOnRefresh: true,
      onTopPassed: function() {
        $header = $(this);
      },
      onTopPassedReverse: function() {
        $header = $(this);
      }
    });
});
function CreateRoom(url) {
  $('.fullscreen.modal').modal('show');
  if (App.token.length <= 0) {
    GetToken(url);
  } else {
    SetFormValueToken(App.token);
  }
}
function AddMessage(url) {
  $('.fullscreen.modal').modal('show');
  $('.ui.radio.checkbox').checkbox();
  if (App.token.length <= 0) {
    GetToken(url);
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
function SubmitForm(elm, url) {
  $(elm).addClass("disabled");
  var data = $(elm).closest('form').serializeArray();
  data = parseJson(data);
  $.ajax({
    type:          'POST',
    url:           url,
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data)
  })
  .always(function() {
    window.setTimeout(() => location.reload(true), 1000);
  });
}
function Report(elm, url) {
  if ($(elm).children('a').text() == 'Reported') {
    return;
  }
  $(elm).addClass('disabled');
  var data = $(elm).closest('form').serializeArray();
  data = parseJson(data);
  $.ajax({
    type:          'POST',
    url:           url,
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data)
  })
  .always(function() {
    $(elm).children('a').text('Reported');
  });
}
function GetToken(url) {
  const data = {action: 'puttoken'};
  $.ajax({
    type:          'POST',
    dataType:      'json',
    contentType:   'application/json',
    scriptCharset: 'utf-8',
    data:          JSON.stringify(data),
    url:           url
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
var App = { token: '' };
