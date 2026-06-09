from collections import Counter


def top_three(texts):
    return [text for text, _ in Counter(texts).most_common(3)]


# print(top_three(['apple', 'banana', 'apple', 'orange', 'banana', 'apple', 'grape', 'grape', 'banana']))
