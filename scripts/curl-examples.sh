#!/bin/bash

BASE_URL="http://localhost:8080"

echo "Aegis Gateway - cURL Examples"
echo "=============================="
echo ""

echo "1. Check Gateway Health:"
echo "   curl $BASE_URL/health"
echo ""

echo "2. Blocked Payment (exceeds limit):"
echo "   curl -H \"X-Agent-ID: finance-agent\" \\"
echo "     -X POST $BASE_URL/tools/payments/create \\"
echo "     -d '{\"amount\":50000,\"currency\":\"USD\",\"vendor_id\":\"V99\"}'"
echo ""

echo "3. Allowed Payment:"
echo "   curl -H \"X-Agent-ID: finance-agent\" \\"
echo "     -X POST $BASE_URL/tools/payments/create \\"
echo "     -d '{\"amount\":1000,\"currency\":\"USD\",\"vendor_id\":\"V42\"}'"
echo ""

echo "4. Refund Payment:"
echo "   curl -H \"X-Agent-ID: finance-agent\" \\"
echo "     -X POST $BASE_URL/tools/payments/refund \\"
echo "     -d '{\"payment_id\":\"PAYMENT_ID_HERE\",\"reason\":\"duplicate\"}'"
echo ""

echo "5. Allowed HR File Read:"
echo "   curl -H \"X-Agent-ID: hr-agent\" \\"
echo "     -X POST $BASE_URL/tools/files/read \\"
echo "     -d '{\"path\":\"/hr-docs/employee-handbook.pdf\"}'"
echo ""

echo "6. Blocked HR File Read (wrong path):"
echo "   curl -H \"X-Agent-ID: hr-agent\" \\"
echo "     -X POST $BASE_URL/tools/files/read \\"
echo "     -d '{\"path\":\"/legal/contract.docx\"}'"
echo ""

echo "7. Write File:"
echo "   curl -H \"X-Agent-ID: accounting-agent\" \\"
echo "     -X POST $BASE_URL/tools/files/write \\"
echo "     -d '{\"path\":\"/accounting/report.txt\",\"content\":\"Q4 data\"}'"
echo ""

echo "8. Reload Policies:"
echo "   curl -X POST $BASE_URL/policies/reload"
echo ""

echo "9. With Parent Agent Header:"
echo "   curl -H \"X-Agent-ID: finance-agent\" \\"
echo "     -H \"X-Parent-Agent: supervisor-agent\" \\"
echo "     -X POST $BASE_URL/tools/payments/create \\"
echo "     -d '{\"amount\":1000,\"currency\":\"USD\",\"vendor_id\":\"V42\"}'"
echo ""
