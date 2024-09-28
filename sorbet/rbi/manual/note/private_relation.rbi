# typed: strict
# frozen_string_literal: true

class Note
  class PrivateRelation
    include ActiveRecordCursorPaginate::Extension
  end
end
