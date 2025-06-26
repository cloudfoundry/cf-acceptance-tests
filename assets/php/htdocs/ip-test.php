<?php

function fetchIpAddress($url) {
    $ch = curl_init($url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_TIMEOUT, 5);
    $response = curl_exec($ch);
    $curlError = curl_error($ch);
    curl_close($ch);

    return [$response, empty($curlError) ? null : $curlError];
}

$target = isset($_GET['target']) ? $_GET['target'] : '';

switch ($target) {
    case 'ipv4':
        $url = 'https://api.ipify.org';
        break;
    case 'ipv6':
        $url = 'https://api6.ipify.org';
        break;
    case 'dual-stack':
        $url = 'https://api64.ipify.org';
        break;
    default:
        header($_SERVER["SERVER_PROTOCOL"] . " 400 Bad Request");
        echo "Error: Invalid target";
        exit;
}

list($response, $curlError) = fetchIpAddress($url);

if ($curlError) {
    header($_SERVER["SERVER_PROTOCOL"] . " 500 Internal Server Error");
    echo "Error: $curlError";
} else {
    echo trim($response);
}

?>