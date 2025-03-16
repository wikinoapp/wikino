# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicNameValidatable
    extend ActiveSupport::Concern

    included do
      validates :name, presence: true

      validate do |record|
        return if space.nil?
        return if name.nil?

        if space.not_nil!.topic_name_uniqueness?(name.not_nil!)
          errors.add(:name, :uniqueness)
        end
      end
    end
  end
end
