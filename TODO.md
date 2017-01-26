restreamer TODO list
====================

Priority
--------

* Implement HA, i.e. multiple sources for streams/resources
* Implement connect/read timeouts for every socket (stream, connection, proxy)
* Add configuration value defaults

Important
---------

* Check concurrent usage of variables and add atomics/locks where appropriate
* Implement load balancing features for HA
* Standardized connection log (JSON?)

Nice2have
---------

* Queue fill rate statistics for more fine-grained debugging
* Add additional configuration variables where appropriate
* Add additional APIs where appropriate (ex. more sophisticated stats)
* Finalize code docs
* Prefill client connection queues
* Support for push streams (UDP/RTP/TCP/...)
