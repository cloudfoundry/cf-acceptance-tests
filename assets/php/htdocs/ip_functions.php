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