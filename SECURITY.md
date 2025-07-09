# Security Policy

## Supported Versions

We actively support the following versions of OAuth2 Server with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| 0.9.x   | :white_check_mark: |
| < 0.9   | :x:                |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in OAuth2 Server, please report it to us privately.

### How to Report

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, please send an email to: **akozadaev@inbox.ru**

Include the following information:
- Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your vulnerability report within 48 hours.
- **Initial Assessment**: We will provide an initial assessment within 5 business days.
- **Regular Updates**: We will keep you informed of our progress throughout the process.
- **Resolution Timeline**: We aim to resolve critical vulnerabilities within 30 days.

### Responsible Disclosure

We follow responsible disclosure practices:

1. **Investigation**: We investigate and verify the reported vulnerability
2. **Fix Development**: We develop and test a fix
3. **Coordinated Release**: We coordinate the release of the fix
4. **Public Disclosure**: We publicly disclose the vulnerability after the fix is released

### Security Best Practices

When using OAuth2 Server in production:

#### 🔒 **Authentication & Authorization**
- Use strong JWT secrets (minimum 32 characters)
- Implement proper token expiration times
- Regularly rotate JWT secrets
- Use HTTPS/TLS for all communications

#### 🛡️ **Infrastructure Security**
- Keep dependencies up to date
- Use secure database configurations
- Implement proper firewall rules
- Regular security audits

#### 📊 **Monitoring & Logging**
- Monitor for suspicious authentication attempts
- Log all security-relevant events
- Set up alerts for unusual patterns
- Regular log analysis

#### 🔄 **Operational Security**
- Regular backups of critical data
- Incident response procedures
- Security training for team members
- Regular penetration testing

### Security Features

OAuth2 Server includes several built-in security features:

- **JWT Token Security**: HMAC-SHA256 signed tokens
- **Token Expiration**: Configurable access and refresh token lifetimes
- **Secure Storage**: PostgreSQL-based token storage with proper indexing
- **Context Timeouts**: Protection against long-running database operations
- **Input Validation**: Comprehensive input validation and sanitization
- **CORS Protection**: Configurable CORS policies
- **Rate Limiting Ready**: Architecture supports rate limiting implementation

### Security Checklist for Production

- [ ] Strong JWT secrets configured
- [ ] HTTPS/TLS enabled
- [ ] Database connections secured
- [ ] Proper CORS policies set
- [ ] Rate limiting implemented
- [ ] Monitoring and alerting configured
- [ ] Regular security updates applied
- [ ] Backup and recovery procedures tested

### Hall of Fame

We recognize security researchers who help improve OAuth2 Server security:

<!-- Security researchers will be listed here after responsible disclosure -->

### Contact

For security-related questions or concerns:
- **Email**: akozadaev@inbox.ru
- **PGP Key**: [Available on request]

---

**Note**: This security policy is subject to change. Please check back regularly for updates.
