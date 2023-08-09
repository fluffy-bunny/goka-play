

$AZURE_RESOURCE_GROUP  = "Herb"
$AZURE_LOCATION = "westus2"
$EVENT_HUBS_NAMESPACE = "herb-event-hub"
$EVENT_HUB_NAME = "cloudevents"
$EVENT_HUB_NAME_DEADLETTER = "cloudevents-deadletter"
$EVENT_HUB_NAME_GROUP_TABLE = "cloudevents-group-table"

$EVENT_HUB_AUTH_RULE_NAME = "RootManageSharedAccessKey"


az eventhubs namespace create --name $EVENT_HUBS_NAMESPACE --resource-group $AZURE_RESOURCE_GROUP  --location $AZURE_LOCATION --enable-kafka true --enable-auto-inflate false

az eventhubs eventhub create --name $EVENT_HUB_NAME --resource-group $AZURE_RESOURCE_GROUP --namespace-name $EVENT_HUBS_NAMESPACE --partition-count 10
az eventhubs eventhub create --name $EVENT_HUB_NAME_DEADLETTER --resource-group $AZURE_RESOURCE_GROUP --namespace-name $EVENT_HUBS_NAMESPACE --partition-count 10
az eventhubs eventhub create --name $EVENT_HUB_NAME_GROUP_TABLE --resource-group $AZURE_RESOURCE_GROUP --namespace-name $EVENT_HUBS_NAMESPACE --partition-count 10

az eventhubs namespace authorization-rule list --resource-group $AZURE_RESOURCE_GROUP --namespace-name $EVENT_HUBS_NAMESPACE

az eventhubs namespace authorization-rule keys list --resource-group $AZURE_RESOURCE_GROUP --namespace-name $EVENT_HUBS_NAMESPACE --name $EVENT_HUB_AUTH_RULE_NAME

