# typed: strict
# frozen_string_literal: true

module FormConcerns
  module PageTitleValidatable
    extend ActiveSupport::Concern

    included do
      validates :title,
        filename_safe: true,
        length: {maximum: Page::TITLE_MAX_LENGTH},
        presence: true
    end
  end
end
