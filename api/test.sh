#! /bin/bash
echo "start testing"

login_jose () {
    curl -X POST -u jose:maria localhost:8080/login | jq;
}
logout_jose() {
    curl -X DELETE \
         -H "Authorization: Bearer am9zZTptYXJpYQ==" \
         localhost:8080/logout | jq
}

while test $# -gt 0; do
  case "$1" in
    --post-workloads)
        login_jose
        curl -H "Content-Type: application/json" \
             -H "Authorization: Bearer am9zZTptYXJpYQ==" \
            -X POST \
            -d '{"filter": "blur", "workload_name": "jose"}' \
            localhost:8080/workloads | jq
        logout_jose
        shift
        ;;
    --get-workloads)
        login_jose
        curl -H "Content-Type: application/json" \
             -H "Authorization: Bearer am9zZTptYXJpYQ==" \
            -X GET \
            localhost:8080/workloads/2 | jq
        logout_jose
        shift
        ;;
    --post-images)
        login_jose
        curl -H "Content-Type: multipart/form-data" \
             -H "Authorization: Bearer am9zZTptYXJpYQ==" \
             -F "data=@test.png" \
             -F "workload_id=0" \
             -F "type=original" \
             -X POST \
             localhost:8080/images | jq
        logout_jose
        shift
        ;;
    --get-images)
        login_jose
        curl -H "Content-Type: application/json" \
             -H "Authorization: Bearer am9zZTptYXJpYQ==" \
            -X GET \
            localhost:8080/images/1 \
            --output file.png
            logout_jose
        shift
        ;;
    --get-status)
        login_jose
        curl -H "Content-Type: application/json" \
             -H "Authorization: Bearer am9zZTptYXJpYQ==" \
            -X GET \
            localhost:8080/status | jq
        logout_jose
        shift
        ;;
    *)
      break
      ;;
  esac
done

