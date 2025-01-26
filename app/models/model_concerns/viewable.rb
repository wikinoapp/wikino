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

    sig { abstract.params(page: Page).returns(T::Boolean) }
    def can_view_page?(page:)
    end

    sig { abstract.params(space: Space).returns(T::Boolean) }
    def can_view_trash?(space:)
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
  end
end
