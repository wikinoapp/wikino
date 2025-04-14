# typed: strict
# frozen_string_literal: true

class MovePageToTrashService < ApplicationService
  class Result < T::Struct
    const :page, PageRecord
  end

  sig { params(page: PageRecord).returns(Result) }
  def call(page:)
    page.touch(:trashed_at)

    Result.new(page:)
  end
end
