# typed: strict
# frozen_string_literal: true

class PageRecord
  class PrivateRelationWhereChain
    include ActiveRecordCursorPaginate::Extension

    def destroy_all_with_related_records!
    end
  end
end
