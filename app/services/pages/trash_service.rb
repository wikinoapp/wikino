# typed: strict
# frozen_string_literal: true

module Pages
  class TrashService < ApplicationService
    class Result < T::Struct
      const :page_record, PageRecord
    end

    sig { params(page_record: PageRecord).returns(Result) }
    def call(page_record:)
      page_record.touch(:trashed_at)

      Result.new(page_record:)
    end
  end
end
