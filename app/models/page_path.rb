# typed: strict
# frozen_string_literal: true

class PagePath < T::Struct
  include T::Struct::ActsAsComparable

  const :topic_name, String
  const :page_title, String
end
