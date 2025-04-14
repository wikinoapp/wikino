# typed: strict
# frozen_string_literal: true

module ModelConcerns
  module Viewable
    extend T::Sig
    extend T::Helpers

    interface!

    sig { abstract.returns(T::Boolean) }
    def signed_in?
    end

    sig { abstract.returns(T.nilable(UserEntity)) }
    def user_entity
    end

    sig { abstract.params(space: Space).returns(ModelConcerns::SpaceViewable) }
    def space_viewer!(space:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def joined_space?(space:)
    end

    sig { abstract.params(topic: Topic).returns(T::Boolean) }
    def can_view_topic?(topic:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_trash_page?(page:)
    end

    sig { abstract.returns(String) }
    def serialized_locale
    end

    sig { abstract.returns(String) }
    def time_zone
    end

    sig { abstract.returns(Topic::PrivateRelation) }
    def viewable_topics
    end

    sig { returns(Locale) }
    def locale
      Locale.deserialize(serialized_locale)
    end
  end
end
