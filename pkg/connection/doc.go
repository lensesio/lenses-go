/*
The Lenses API has two groups of endpoints to manage connections: a "generic"
one and several "specific" ones. The generic one accepts all possible
connection objects. Such an object has a "templateName" which is the type of the
object, e.g.: PostgreSQL, Elasticsearch, etc. There are also a few specific
endpoints that accept one connection "flavour". Examples are Kafka,
KafkaConnect, etc. The generic API is a superset of the specific ones: every
specific connection can be administered via the generic API, only a handful
specific connections exist.

The generic endpoint is: /api/v1/connection/connections;
An example of a specific endpoint is: /api/v1/connection/connection-templates/KafkaConnect.
*/
package connection
