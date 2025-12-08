# typed: strict
# frozen_string_literal: true

module FormConcerns
  module SpaceNameValidatable
    extend ActiveSupport::Concern

    included do
      validates :name,
        length: {maximum: Space::NAME_MAX_LENGTH},
        presence: true
    end
  end
end
