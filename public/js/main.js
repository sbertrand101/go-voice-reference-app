var authData = JSON.parse(localStorage.getItem('authData') || '{}');

var loginForm = document.getElementById('loginForm');
var registerForm = document.getElementById('registerForm');
var phoneContainer = document.getElementById('phone');

var screens = [loginForm, registerForm, phoneContainer];

loginForm.getElementsByTagName('a')[0].addEventListener('click', function(e){
	e.preventDefault();
	switchToScreen(registerForm);
});

loginForm.getElementsByTagName('form')[0].addEventListener('submit', function(e){
	e.preventDefault();
	var button = e.target.getElementsByTagName('button')[0];
	button.setAttribute('disabled', 'disabled');
	login(e.target.elements['userName'].value, e.target.elements['password'].value)
	.then(saveAuthData, function(err){
		setError(loginForm, err);
		button.removeAttribute('disabled');
	});
});

registerForm.getElementsByTagName('form')[0].addEventListener('submit', function(e){
	var fields = e.target.elements;
	e.preventDefault();
	var button = e.target.getElementsByTagName('button')[0];
	button.setAttribute('disabled', 'disabled');
	fetch('/register', {
		method: 'POST',
		headers: {
			'Accept': 'application/json',
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({
			userName: fields['userName'].value,
			areaCode: fields['areaCode'].value,
			password: fields['password'].value,
			repeatPassword: fields['repeatPassword'].value
		})
	})
	.then(checkResponse)
	.then(function(){
		return login(fields['userName'].value, fields['password'].value);
	})
	.then(saveAuthData, function(err){
		setError(registerForm, err);
		button.removeAttribute('disabled');
	});
});


function login(userName, password) {
	return fetch('/login', {
		method: 'POST',
		headers: {
			'Accept': 'application/json',
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({userName: userName, password: password})
	})
	.then(checkResponse)
}

function saveAuthData(body) {
	authData = body;
	localStorage.setItem('authData', JSON.stringify(authData));
	switchToScreen(phoneContainer);
}

function switchToScreen(screen) {
	screens.forEach(function(s){
		s.setAttribute('hidden', 'hidden');
	});
	screen.removeAttribute('hidden');
}

function setError(element, err) {
	var error = element.getElementsByClassName('error')[0];
	if (err) {
		error.textContent = err.message || err;
		error.removeAttribute('hidden');
	}
	else {
		error.setAttribute('hidden', 'hidden');
	}
}

function checkResponse(result) {
	return result.json().then(function(body){
		if (result.ok) {
			return body;
		}
		var err =  new Error(body.message)
		err.code = body.code;
		throw err;
	});
}

document.addEventListener('DOMContentLoaded', function(){
	switchToScreen(loginForm);
})
