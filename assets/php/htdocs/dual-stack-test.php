<?php

require 'ip_functions.php';

$url = 'https://api64.ipify.org?format=json';
list($response, $curlError) = fetchIpAddress($url);

if ($response === false) {
    $validationType = "Dual-stack";
    $message = createResponseMessage($validationType, false, "unknown", $curlError);
    echo $message;
    exit;
}

$data = json_decode($response, true);
$ip = $data['ip'] ?? null;

$ipType = determineIpType($ip);
$isSuccess = $ipType === "IPv4" || $ipType === "IPv6";

$validationType = "Dual stack";
$message = createResponseMessage($validationType, $isSuccess, $ipType, $isSuccess ? "none" : "failure");
echo $message;

?>