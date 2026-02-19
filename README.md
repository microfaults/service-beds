# service-beds
Contains deployable services as a mesh that provides certain transactional flows

## Online-Boutique
1. Ported all services to Go.
2. Extensions (TODO):
   - Add queue between checkout and others (gives feedback ideas down the line if persistence is used later) or add Kafka just after email service for simplicity
   - Add DB writes after checkouts, let's use TTL of 30 days or some MAX_ENTRIES variable beyond which SQL cleanup is triggered
   - Scale up number of products, high res listings, etc.