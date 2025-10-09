TF_ACC=1 \
TF_ACC_PROJECT_ID="571e38eb-c04f-4842-afe0-470781b4b13e" \
STACKIT_SERVICE_ACCOUNT_KEY_PATH="/Users/correia_poli/Workspace/terraform-provider-stackit/tmp/sa-key-00397968-db14-4e17-ad0b-d5b2207a93dc.json" \
TF_ACC_REGION="eu01" \
go test -timeout=20m -v ./stackit/internal/services/cdn/cdn_acc_test.go