<?php

function determineIpType($ip) {
    if (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV4)) {
        return "IPv4";
    } elseif (filter_var($ip, FILTER_VALIDATE_IP, FILTER_FLAG_IPV6)) {
        return "IPv6";
    }
    return "Invalid IP";
}

function createResponseMessage($validationType, $isSuccess, $ipType, $errorMessage = "none") {
    $status = $isSuccess ? "success" : "failure";
    return "$validationType validation resulted in $status. Detected IP type is $ipType. Error message: $errorMessage.";
}

function fetchIpAddress($url) {
    $curlHandle = curl_init($url);
    curl_setopt_array($curlHandle, [
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_FAILONERROR => true,
        CURLOPT_TIMEOUT => 5,
    ]);

    $response = curl_exec($curlHandle);
    $error = curl_error($curlHandle);
    curl_close($curlHandle);

    return [$response, $error];
}

?>