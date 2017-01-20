restreamer TODO list
====================

Priority
--------

* Finish implementation of static content server/proxy
* Implement HA, i.e. multiple sources for streams/resources
* Separate health checks and statistics
* Implement global limits

Important
---------

* Check concurrent usage of variables and add atomics/locks where appropriate
* Implement load balancing features for HA

Nice2have
---------

* Add additional configuration variables where appropriate
* Add additional APIs where appropriate (ex. more sophisticated stats)
* Finalize code docs
