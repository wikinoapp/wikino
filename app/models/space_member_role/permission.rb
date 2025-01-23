# typed: strict
# frozen_string_literal: true

module SpaceMemberRole
  class Permission < T::Enum
    enums do
      CreateTopic = new("create:topic")
      CreatePage = new("create:page")
      CreateDraftPage = new("create:draft_page")
    end
  end
end
