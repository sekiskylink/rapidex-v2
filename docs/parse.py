def parse_message(msg, kw=""):
    msg = msg.strip().lower()
    msg = msg.replace(',', '.')
    msg = msg.replace(' ', '.')
    separators = []
    segments = []
    separator = DELIMITER or '\\s'
    search_str = '[%s]+' % separator
    match = re.search(search_str, msg)
    while match:
        segment = msg[0:match.start()]
        separator = msg[match.start():match.end()]
        segments.append(segment)
        separators.append(separator)
        msg = msg[match.end():]
        match = re.search(search_str, msg)
    segments.append(msg)

    # remove empty segments
    stripped_segments = []
    for segment in segments:
        segment = segment.strip()
        if len(segment):
            stripped_segments.append(segment)

    segments = stripped_segments
    # do more cleaning here for stuff like ma2 instead of ma.2
    # get stuff like ma2, split it accordingly and put it back in right position
    outliers = [x for x in segments if not x.isalpha() and not x.isdigit()]
    for i in outliers:
        idx = segments.index(i)
        match_num = re.search('[0-9]', i)
        if match_num:
            command_segment = i[0:match_num.start()]
            num_segment = i[match_num.start():match_num.end()]
            segments.insert(idx, command_segment)
            segments[idx + 1] = num_segment

    if not len(segments) % 2 == 0:
        return "not all commands have a value"
    command_value_pairs = []
    positions = DEATH_POSITIONS if kw == 'death' else CASES_POSITIONS
    dummy_resp = ['0' for i in range(KEYWORDS_DATA_LENGTH[kw])]

    for idx, segment in enumerate(segments):
        if segment in positions:
            cmd = segment
            if (idx + 1) <= (len(segments) - 1):
                val = segments[idx + 1]
                dummy_resp[positions[segment]] = val
            else:
                # return an error
                val = 0
            command_value_pairs.append((cmd, val))
        else:
            continue

    return '.'.join(dummy_resp)
