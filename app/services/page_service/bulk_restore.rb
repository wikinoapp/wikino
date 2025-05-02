# typed: strict
# frozen_string_literal: true

module PageService
  class BulkRestore < ApplicationService
    class Result < T::Struct
      const :pages, PageRecord::PrivateRelation
    end

    sig { params(page_ids: T::Array[T::Wikino::DatabaseId]).returns(Result) }
    def call(page_ids:)
      pages = PageRecord.where(id: page_ids)

      pages.update_all(
        trashed_at: nil,
        updated_at: Time.current
      )

      Result.new(pages:)
    end
  end
end
