# typed: strict
# frozen_string_literal: true

UUID_FORMAT = T.let(/[0-9a-f]{8}-[0-9a-f]{4}-[0-5][0-9a-f]{3}-[089ab][0-9a-f]{3}-[0-9a-f]{12}/i.freeze, Regexp)
