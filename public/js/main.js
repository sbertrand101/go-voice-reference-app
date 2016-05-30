(function() {
	JsSIP.debug.enable('JsSIP:*');

	var missingFeatures = [];

	if (!JsSIP.rtcninja.hasWebRTC()) {
		missingFeatures.push("WebRTC");
	}

	if (!window.WebSocket) {
		missingFeatures.push("WebSockets");
	}

	if (!window.fetch) {
		var fetchApi = document.createElement('script');
		fetchApi.setAttribute('src', '//cdnjs.cloudflare.com/ajax/libs/fetch/1.0.0/fetch.min.js');
		document.getElementsByTagName('head')[0].appendChild(fetchApi);
	}


	HTMLElement.prototype.show = function(){
		this.removeAttribute('hidden');
	}

	HTMLElement.prototype.hide = function(){
		this.setAttribute('hidden', 'hidden');
	}

	var authData = JSON.parse(localStorage.getItem('authData') || '{}');

	var loginForm = document.getElementById('loginForm');
	var registerForm = document.getElementById('registerForm');
	var phoneContainer = document.getElementById('phone');
	var connecting = document.getElementById('connecting');
	var unsupportedBrowser = document.getElementById('unsupportedBrowser');

	var screens = [loginForm, registerForm, connecting, phoneContainer, unsupportedBrowser];

	var incomingCallAudio = document.getElementById('incomingCallAudio');
	var remoteAudio = document.getElementById('remoteAudio');
	var toField = document.getElementById('toField');
	var phoneNumber = document.getElementById('phoneNumber');
	var sipDetails = document.getElementById('sipDetails');
	var sipDetailsLink = document.getElementById('sipDetailsLink');
	var voiceMailLink = document.getElementById('voiceMailLink');
	var voiceMailMessages = document.getElementById('voiceMailMessages');
	var noVoiceMailMessages = document.getElementById('noVoiceMailMessages');
	var voiceMailMessagesList = voiceMailMessages.getElementsByTagName('ul')[0];
	var sipUserName = document.getElementById('sipUserName');
	var sipDomain = document.getElementById('sipDomain');
	var sipPassword = document.getElementById('sipPassword');
	var logOut = document.getElementById('logOut');

	var phone, session, sipAuthHeader, callOptions;

	if (missingFeatures.length > 0) {
		unsupportedBrowser.getElementsByClassName('features')[0].innerHTML = missingFeatures.join(', ');
		switchToScreen(unsupportedBrowser);
		return;
	}

	loginForm.getElementsByTagName('a')[0].addEventListener('click', function(e){
		e.preventDefault();
		switchToScreen(registerForm);
	});

	registerForm.getElementsByTagName('a')[0].addEventListener('click', function(e){
		e.preventDefault();
		switchToScreen(loginForm);
	});

	loginForm.getElementsByTagName('form')[0].addEventListener('submit', function(e){
		e.preventDefault();
		var button = e.target.getElementsByTagName('button')[0];
		button.setAttribute('disabled', 'disabled');
		login(e.target.elements['userName'].value, e.target.elements['password'].value)
		.then(saveAuthData)
		.then(start, function(err){
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
		.then(saveAuthData)
		.then(start, function(err){
			setError(registerForm, err);
			button.removeAttribute('disabled');
		});
	});

	voiceMailMessages.getElementsByClassName('change-greeting')[0].addEventListener('click', function(e){
		e.preventDefault();
		fetch('/recordGreeting', {
			method: 'POST',
			headers: {
				'Authorization': 'Bearer ' + authData.token
			}
		})
		.then(checkResponse, function(err){
			setError(document, err);
		});
	});

	function makeCall(){
		var number = toField.value;
		if (!number) {
			return;
		}
		if (number.length === 10) {
			number = '+1' + number;
		}
		phone.call(number, callOptions);
		updateDialerUI();
	}

	function addVoiceMailMessage(msg) {
		var item = document.createElement('li');
		var startTime = new Date(msg.startTime);
		item.innerHTML = '<a href="#">' + msg.from + ' - ' + startTime.toLocaleString('en') +' (' + (new Date(msg.endTime) - new Date(msg.startTime))/1000 + 's)</a>' +
			'<a href="#" class="remove"><span class="fa fa-trash"></span></a>';
		var remove = item.getElementsByClassName('remove')[0];
		remove.addEventListener('click', function(e){
			e.preventDefault();
			fetch('/voiceMessages/' + msg.id, {
				method: 'DELETE',
				headers: {
					'Authorization': 'Bearer ' + authData.token
				}
			})
			.then(function(r){
				if (r.ok) {
					return item.remove();
				}
				throw new Error('Error on removing message');
			}, function(err){
				setError(document, err);
			})
		});
		item.getElementsByTagName('a')[0].addEventListener('click', function(e){
			e.preventDefault();
			fetch('/voiceMessages/' + msg.id + '/media', {
				headers: {
					'Authorization': 'Bearer ' + authData.token
				}
			})
			.then(function(r){
				if (r.ok) {
					return r.blob();
				}
				throw new Error('Error on downloading file');
			})
			.then(function(blob){
				// download loaded data as file
				var link = document.createElement('a');
				link.setAttribute('href', window.URL.createObjectURL(blob));
				link.click();
			}, function(err){
				setError(document, err);
			})
		});
		return item;
	}

	document.getElementById('connectCall').addEventListener('click', makeCall);

	document.getElementById('answer').addEventListener('click', function(){
		session.answer(callOptions);
	});

	document.getElementById('hangUp').addEventListener('click', hangup);
	document.getElementById('reject').addEventListener('click', hangup);

	document.getElementById('mute').addEventListener('click', function(e){
		e.preventDefault();
		console.log('MUTE CLICKED');
		if(session.isMuted().audio){
			session.unmute({audio: true});
		}
		else{
			session.mute({audio: true});
		}
		updateDialerUI();
	});

	sipDetailsLink.addEventListener('click', function(){
		sipDetailsLink.style.display = 'none';
		sipDetails.show();
	});

	voiceMailLink.addEventListener('click', function(){
		voiceMailLink.style.display = 'none';
		voiceMailMessages.show();
	});

	logOut.addEventListener('click', makeLogOut);

	toField.addEventListener('keypress', function(e){
		if(e.which === 13){//enter
			makeCall();
		}
	});

	var i, buttons = document.getElementById('inCallButtons').getElementsByClassName('dialpad-char');
	for(i = 0; i < buttons.length; i ++) {
		buttons[i].addEventListener('click', function (e) {
			e.preventDefault();
			var digit = e.target.getAttribute('data-value');
			console.log("Send DTMF: " + digit);
			session.sendDTMF(digit);
		});
	};

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
	}

	function switchToScreen(screen) {
		screens.forEach(function(s){
			s.hide();
		});
		if (screen) {
			screen.show();
		}
	}

	function setError(element, err) {
		var error = element.getElementsByClassName('error')[0];
		if (err) {
			error.textContent = err.message || err;
			error.show();
		}
		else {
			error.hide();
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

	function authed(fn) {
		if (!authData.expire || !authData.token) {
			return switchToScreen(loginForm);
		}
		return refreshAuthData().then(function(){
			logOut.show();
			fn();
		}, makeLogOut);
	}

	function makeLogOut() {
		localStorage.removeItem('authData')
		window.location.reload();
	}

	function refreshAuthData() {
		return fetch('/refreshToken', {
			headers: {
				'Authorization': 'Bearer ' + authData.token
			}
		})
		.then(checkResponse)
		.then(saveAuthData);
	}

	function setupPhone(sipData) {
		var tries = 15;
		sipAuthHeader = 'X-Callsign-Token: ' + sipData.token;
		callOptions = {
			extraHeaders: [sipAuthHeader],
			mediaConstraints: {
				audio: true,
				video: false
			}
		};
		if (phone) {
			phone.stop();
		}
		phone = new JsSIP.UA({
			'uri': sipData.sipUri,
			'ws_servers': 'wss://webrtc.registration.bandwidth.com:10443'
		});
		phone.registrator().setExtraHeaders([sipAuthHeader]);

		phone.on('registered', function(){
			switchToScreen(phoneContainer);
			toField.focus();
			setError(document, null);
		});

		phone.on('registrationFailed', function(e){
			if ((--tries) > 0) {
				setTimeout(function(){
					phone.register()
				}, 10000); // try to reregister again
				return;
			}
			setError(document, e.cause);
			switchToScreen(null);
		});

		phone.on('newRTCSession', function(data){
			setSession(data.session);
		});
		switchToScreen(connecting);
		phone.sipData = sipData;
		phone.start();
		phoneNumber.innerHTML = sipData.phoneNumber;
		sipPassword.innerHTML = sipData.sipPassword;
		var m = /sip\:([\w\.\-_]+)@([\w\.\-_]+)/i.exec(sipData.sipUri);
		sipUserName.innerHTML = m[1];
		sipDomain.innerHTML = m[2];
	}

	function setVoiceMessages(messages) {
		(messages || []).forEach(function(msg){
			voiceMailMessagesList.appendChild(addVoiceMailMessage(msg));
		});
		if (messages.length > 0) {
			noVoiceMailMessages.hide();
			voiceMailMessagesList.show();
		}
		else {
			noVoiceMailMessages.show();
			voiceMailMessagesList.hide();
		}
		if (!window.EventSource) { // if SSE supported
			return;
		}
		// show new voice messages in real time (without reloading of page)
		var source = new window.EventSource('/voiceMessagesStream?token=' + authData.token);
		source.addEventListener('message', function(e){
			var msg = JSON.parse(e.data);
			var item = addEventListener(msg);
			voiceMailMessages.insertBefore(item, voiceMailMessages.firstChild);
			item.scrollIntoView();
			noVoiceMailMessages.hide();
			voiceMailMessagesList.show();
		});
	}

	function setSession(s) {
		if (session === s) {
			return;
		}
		incomingCallAudio.pause();
		hangup();
		session = s;
		if (session) {
			session.on('ended', function(){
				setSession(null);
			});
			session.on('failed', function(){
				setSession(null);
			});
			session.on('confirmed', function(){
				updateDialerUI();
			});
			session.on('addstream', function(e){
				incomingCallAudio.pause();
				remoteAudio.src = window.URL.createObjectURL(e.stream);
			});
		}
		updateDialerUI();
	}

	function hangup() {
		if (session && !session.isEnded()) {
			session.terminate({extraHeaders: [sipAuthHeader]});
		}
	}

	function updateDialerUI(){
		if(session){
			if(session.isInProgress()){
				if(session.direction === 'incoming'){
					incomingCallAudio.play();
					document.getElementById('incomingCallNumber').innerHTML = session.remote_identity.uri.user;
					document.getElementById('incomingCall').show();
					document.getElementById('callControl').hide();
					document.getElementById('incomingCall').show();
				}else{
					document.getElementById('callInfoText').innerHTML = 'Ringing...';
					document.getElementById('callInfoNumber').innerHTML = session.remote_identity.uri.user;
					document.getElementById('callStatus').show();
				}

			}else if(session.isEstablished()){
				document.getElementById('callStatus').show();
				document.getElementById('incomingCall').hide();
				document.getElementById('callInfoText').innerHTML = 'In Call';
				document.getElementById('callInfoNumber').innerHTML = session.remote_identity.uri.user;
				document.getElementById('inCallButtons').show();
				incomingCallAudio.pause();
			}
			document.getElementById('callControl').hide();
		}else{
			document.getElementById('incomingCall').hide();
			document.getElementById('callControl').show();
			document.getElementById('callStatus').hide();
			document.getElementById('inCallButtons').hide();
			incomingCallAudio.pause();
		}
		//microphone mute icon
		var muteIcon = document.getElementById('muteIcon');
		if(session && session.isMuted().audio){
			muteIcon.classList.add('fa-microphone-slash');
			muteIcon.classList.remove('fa-microphone');
		}else{
			muteIcon.classList.remove('fa-microphone-slash');
			muteIcon.classList.add('fa-microphone');
		}
	}

	function start() {
		authed(function(){
			var get = function(url) {
				return fetch(url, {
					headers: {
						'Authorization': 'Bearer ' + authData.token
					}
				})
				.then(checkResponse);
			};
			Promise.all([get('/sipData').then(setupPhone), get('/voiceMessages').then(setVoiceMessages)])
			.catch(function(err) {
				setError(document, err);
			});
		});
	}
	document.addEventListener('DOMContentLoaded', start);
})();
