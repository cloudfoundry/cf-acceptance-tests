<?php

require 'ip_functions.php';

$url = 'https://api.ipify.org?format=json';
list($response, $curlError) = fetchIpAddress($url);

if ($response === false) {
    $validationType = "IPv4";
    $message = createResponseMessage($validationType, false, "unknown", $curlError);
    echo $message;
    exit;
}

$data = json_decode($response, true);
$ip = $data['ip'] ?? null;

$ipType = determineIpType($ip);
$isSuccess = $ipType === "IPv4";

$validationType = "IPv4";
$message = createResponseMessage($validationType, $isSuccess, $ipType, $isSuccess ? "none" : "failure");
echo $message;

?>