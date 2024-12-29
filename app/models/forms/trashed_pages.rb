# typed: strict
# frozen_string_literal: true

module Forms
  class TrashedPages < Forms::Base
    sig { returns(T.nilable(T::Array[T::Wikino::DatabaseId])) }
    attr_accessor :page_ids

    validates :page_ids, presence: true
  end
end
