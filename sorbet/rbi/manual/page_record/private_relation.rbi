# typed: strict
# frozen_string_literal: true

class PageRecord
  class PrivateRelation
    include ActiveRecordCursorPaginate::Extension
  end
end
