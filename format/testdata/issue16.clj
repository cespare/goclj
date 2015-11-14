(cond
  (string? x) (try
                (Long/parseLong x)))
