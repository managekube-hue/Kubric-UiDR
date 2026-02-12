/**
 * Structured Data for SEO
 * Adds JSON-LD schema markup for rich snippets in search results
 */

(function() {
  try {
    // Organization Schema
    const organizationSchema = {
      '@context': 'https://schema.org',
      '@type': 'Organization',
      'name': 'Kubric',
      'url': 'https://kubric-platform.vercel.app',
      'logo': 'https://kubric-platform.vercel.app/img/kubric-logo.svg',
      'description': 'Enterprise Security Operations & Orchestration Platform',
      'foundingDate': '2024',
      'sameAs': [
        'https://github.com/managekube-hue/Kubric-UiDR',
        'https://twitter.com/kubric'
      ],
      'contactPoint': {
        '@type': 'ContactPoint',
        'contactType': 'Customer Support',
        'email': 'support@kubric.io'
      }
    };

    // Product/Thing Schema
    const productSchema = {
      '@context': 'https://schema.org',
      '@type': 'SoftwareApplication',
      'name': 'Kubric Enterprise Platform',
      'applicationCategory': 'SecurityApplication',
      'description': 'Unified platform for SOC, NOC, GRC, PSA, and intelligent orchestration',
      'version': '1.0.0',
      'url': 'https://kubric-platform.vercel.app',
      'downloadUrl': 'https://github.com/managekube-hue/Kubric-UiDR',
      'operatingSystem': 'Linux, Ubuntu',
      'abstract': 'Enterprise-grade unified platform for Security Operations Center (SOC), Network Operations Center (NOC), Governance Risk Compliance (GRC), Professional Services Automation (PSA), and AI-powered orchestration with CrewAI',
      'author': {
        '@type': 'Organization',
        'name': 'ManageKube'
      },
      'offers': {
        '@type': 'Offer',
        'price': '0',
        'priceCurrency': 'USD'
      }
    };

    // CrewAI capabilities
    const aiCapabilitiesSchema = {
      '@context': 'https://schema.org',
      '@type': 'Thing',
      'name': 'AI Capabilities',
      'description': 'CrewAI-powered autonomous agents for security and operations',
      'subjectOf': [
        {
          '@type': 'Thing',
          'name': 'Triage Agent',
          'description': 'Incident analysis and KISS calculation'
        },
        {
          '@type': 'Thing',
          'name': 'Housekeeper Agent',
          'description': 'Drift detection and Ansible remediation'
        },
        {
          '@type': 'Thing',
          'name': 'Billing Agent',
          'description': 'Usage tracking and invoicing'
        },
        {
          '@type': 'Thing',
          'name': 'Comms Agent',
          'description': 'VAPI phone and Twilio SMS integration'
        }
      ]
    };

    // Breadcrumb Schema (dynamic, populated on each page)
    function createBreadcrumbSchema() {
      const pathParts = window.location.pathname.split('/').filter(p => p);
      const breadcrumbs = [];
      let currentPath = '';

      pathParts.forEach((part, index) => {
        currentPath += '/' + part;
        const label = part
          .replace(/-/g, ' ')
          .replace(/([A-Z])/g, ' $1')
          .trim()
          .replace(/\b\w/g, l => l.toUpperCase());
        
        breadcrumbs.push({
          '@type': 'ListItem',
          'position': index + 1,
          'name': label,
          'item': 'https://kubric-platform.vercel.app' + currentPath
        });
      });

      if (breadcrumbs.length > 0) {
        return {
          '@context': 'https://schema.org',
          '@type': 'BreadcrumbList',
          'itemListElement': breadcrumbs
        };
      }
      return null;
    }

    // WebPage for current page
    function createWebPageSchema() {
      const title = document.title || 'Kubric Enterprise Platform';
      const description = document.querySelector('meta[name="description"]')?.content || 
                         'Enterprise Security Operations & Orchestration Platform';

      return {
        '@context': 'https://schema.org',
        '@type': 'WebPage',
        'name': title,
        'description': description,
        'url': window.location.href,
        'mainEntity': {
          '@type': 'Article',
          'headline': title,
          'description': description,
          'author': {
            '@type': 'Organization',
            'name': 'Kubric'
          }
        },
        'isPartOf': {
          '@type': 'WebSite',
          'name': 'Kubric Documentation',
          'url': 'https://kubric-platform.vercel.app'
        }
      };
    }

    // Inject all schemas
    function injectSchemas() {
      const schemas = [
        organizationSchema,
        productSchema,
        aiCapabilitiesSchema,
        createBreadcrumbSchema(),
        createWebPageSchema()
      ];

      schemas.forEach(schema => {
        if (schema) {
          const script = document.createElement('script');
          script.type = 'application/ld+json';
          script.textContent = JSON.stringify(schema);
          document.head.appendChild(script);
        }
      });

      console.log('✅ Structured data schemas injected for SEO');
    }

    // Wait for DOM to be ready
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', injectSchemas);
    } else {
      injectSchemas();
    }

  } catch (error) {
    console.error('❌ Error injecting structured data:', error);
  }
})();
