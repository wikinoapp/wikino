# typed: strict
# frozen_string_literal: true

module FormConcerns
  module TopicVisibilityValidatable
    extend ActiveSupport::Concern

    included do
      validates :visibility,
        inclusion: {
          # TODO: 非公開トピックの作成を有料プランでのみ利用可能にする
          in: TopicVisibility.values.map(&:serialize) - [TopicVisibility::Private.serialize]
        },
        presence: true
    end
  end
end
