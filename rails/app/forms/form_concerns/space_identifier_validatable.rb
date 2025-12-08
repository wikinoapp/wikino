# typed: strict
# frozen_string_literal: true

module FormConcerns
  module SpaceIdentifierValidatable
    extend ActiveSupport::Concern

    included do
      validates :identifier,
        exclusion: {in: Space::IDENTIFIER_RESERVED_WORDS, message: :reserved},
        format: {with: Space::IDENTIFIER_FORMAT},
        length: {maximum: Space::IDENTIFIER_MAX_LENGTH},
        presence: true
    end
  end
end
