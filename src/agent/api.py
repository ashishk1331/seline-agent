import requests as R


def fetch(url, headers, payload):
    resp = R.post(
        url,
        headers=headers,
        json=payload,
    )

    if resp.status_code != 200:
        print(
            f"[ERROR] API request failed with status code {resp.status_code}: {resp.text}"
        )
        return None

    return resp.json()
