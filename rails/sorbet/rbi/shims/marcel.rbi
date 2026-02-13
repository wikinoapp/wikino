# typed: strict
# frozen_string_literal: true

module Marcel
  class MimeType
    sig { params(pathname_or_io: T.any(String, IO, StringIO), name: T.nilable(String), magic: T.nilable(T::Boolean), declared_type: T.nilable(String)).returns(String) }
    def self.for(pathname_or_io, name: nil, magic: nil, declared_type: nil); end
  end
end