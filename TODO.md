restreamer TODO list
====================

Priority
--------

* Finish implementation of static content server/proxy
  https://developers.google.com/web/fundamentals/performance/optimizing-content-efficiency/http-caching
* Implement HA, i.e. multiple sources for streams/resources
* Separate health checks and statistics (metrics need to be defined!)
* Implement global limits
* Implement connect/read timeouts for every socket (stream, connection, proxy)
* Add configuration value defaults

Important
---------

* Check concurrent usage of variables and add atomics/locks where appropriate
* Implement load balancing features for HA
* Standardized connection log (JSON?)

Nice2have
---------

* Add additional configuration variables where appropriate
* Add additional APIs where appropriate (ex. more sophisticated stats)
* Finalize code docs
* Prefill client connection queues
