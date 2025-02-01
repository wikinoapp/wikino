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

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def joined_space?(space:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_view_page?(page:)
    end

    sig { abstract.params(topic: Topic).returns(T::Boolean) }
    def can_view_topic?(topic:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_view_trash?(space:)
    end

    sig { abstract.params(topic: Topic).returns(T::Boolean) }
    def can_create_topic?(topic:)
    end

    sig { abstract.params(topic: Topic).returns(T::Boolean) }
    def can_create_page?(topic:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_create_bulk_restored_pages?(space:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_update_draft_page?(page:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_update_page?(page:)
    end

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_trash_page?(page:)
    end

    sig { abstract.returns(String) }
    def time_zone
    end

    sig { abstract.returns(ViewerLocale) }
    def locale
    end

    sig { abstract.returns(Topic::PrivateRelation) }
    def viewable_topics
    end

    sig { abstract.returns(T.any(DraftPage::PrivateAssociationRelation, DraftPage::PrivateRelation)) }
    def active_draft_pages
    end

    sig { abstract.params(space: Space, number: T.untyped).returns(Topic) }
    def find_topic_by_number!(space:, number:)
    end
  end
end
