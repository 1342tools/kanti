[![Contributors][contributors-shield]][contributors-url]
[![Stargazers][stars-shield]][stars-url]
[![Forks][forks-shield]][forks-url]
[![MIT License][license-shield]][license-url]

<br />
<div align="center">
  <a href="https://github.com/othneildrew/Best-README-Template">
    <img src="https://github.com/user-attachments/assets/4f8c807a-95e7-4a4a-be61-ff8f7a591313" alt="Logo" width="160" height="80">
  </a>

  <h3 align="center">A web application testing tool built for capturing and modifying http/https requests.</h3>

</div>




![image](https://github.com/user-attachments/assets/022d9a59-1636-48ef-bc3e-a7596a8ddde8)


## Features

- **Proxy**: Intercept, inspect, and modify HTTP/S traffic between your browser and target applications
- **Fuzzer**: Perform customized automated attacks with parameterized payloads (requires ffuf)
- **Repeater**: Manually craft and replay HTTP requests with real-time response analysis
- **Decoder**: Encode/decode data using various schemes (URL, Base64, JWT, etc.)
- **Sitemap**: Visualize collected subdomains and paths

## Installation

### System Requirements/Compatibility

Available for Windows and Linux

Fuzzing tab requires ffuf to be installed and available via PATH

### Installation Steps

Install from releases to get started

If you are interested in modifying the source code you can clone the repository and run these commands to get started

```
#install dependencies
npm ci

#run the application
npm run start

#package executable

npm run svelte-build
npm run package
```

## Quick Start Guide

1. Launch Kanti and create a new project(or open without one)
2. Start the proxy in Settings > Proxy Settings
3. Configure your browser to use Kanti's proxy (default: 127.0.0.1:8080)
4. Browse your target application to populate requests in the application

## Contributing/Security

Contact @kusonooyasumi on X

[contributors-shield]: https://img.shields.io/github/contributors/1342Tools/kanti.svg?style=for-the-badge
[contributors-url]: https://github.com/1342Tools/kanti/graphs/contributors
[stars-shield]: https://img.shields.io/github/stars/1342Tools/kanti.svg?style=for-the-badge
[stars-url]: https://github.com/1342Tools/kanti/stargazers
[license-shield]: https://img.shields.io/github/license/1342Tools/kanti.svg?style=for-the-badge
[license-url]: https://github.com/1342Tools/kanti/blob/main/LICENSE
[forks-shield]: https://img.shields.io/github/forks/1342Tools/kanti.svg?style=for-the-badge
[forks-url]: https://github.com/1342Tools/kanti/network/members
