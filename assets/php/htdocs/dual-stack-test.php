<?php

require 'ip_functions.php';

$url = 'https://api64.ipify.org';
list($response, $curlError) = fetchIpAddress($url);

if ($curlError) {
    header($_SERVER["SERVER_PROTOCOL"] . " 500 Internal Server Error");
    echo "Error: $curlError";
} else {
    echo trim($response);
}

?>