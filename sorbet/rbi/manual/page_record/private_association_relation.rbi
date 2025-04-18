# typed: strict
# frozen_string_literal: true

class PageRecord
  class PrivateAssociationRelation
    include ActiveRecordCursorPaginate::Extension
  end
end
